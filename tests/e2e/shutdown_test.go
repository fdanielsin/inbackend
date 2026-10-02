package e2e

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runningServer struct {
	cmd    *exec.Cmd
	output *syncBuffer
	addr   string
	exited chan error
}

// startServer runs the real binary on a free port and waits until /health answers.
func startServer(t *testing.T, settings ...string) *runningServer {
	t.Helper()
	pg.Require(t)
	port := freePort(t)
	s := &runningServer{output: &syncBuffer{}, addr: "127.0.0.1:" + port, exited: make(chan error, 1)}
	s.cmd = exec.CommandContext(t.Context(), buildServer(t)) //nolint:gosec // G204: runs the binary built by buildServer
	s.cmd.Env = serverEnv(append([]string{"DATABASE_URL=" + pg.NewEmptyDatabase(t), "PORT=" + port}, settings...)...)
	s.cmd.Stdout = s.output
	s.cmd.Stderr = s.output
	require.NoError(t, s.cmd.Start())
	go func() { s.exited <- s.cmd.Wait() }()
	t.Cleanup(func() { _ = s.cmd.Process.Kill() })

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://" + s.addr + "/health") //nolint:noctx // polling a local test server
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 20*time.Second, 100*time.Millisecond, "the server never answered:\n%s", s.output.String())
	return s
}

// waitExit waits for the process to stop by itself and returns its exit code.
func (s *runningServer) waitExit(t *testing.T, within time.Duration) int {
	t.Helper()
	select {
	case err := <-s.exited:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		require.NoError(t, err)
		return 0
	case <-time.After(within):
		t.Fatalf("the server did not stop within %s:\n%s", within, s.output.String())
		return -1
	}
}

// GIVEN a running server / WHEN it receives SIGTERM or SIGINT / THEN it stops cleanly: exit 0 and "Server stopped"
func TestShutdown_OnSignal_ExitsWithCode0(t *testing.T) {
	for name, signal := range map[string]syscall.Signal{"SIGTERM": syscall.SIGTERM, "SIGINT": syscall.SIGINT} {
		t.Run(name, func(t *testing.T) {
			s := startServer(t)

			require.NoError(t, s.cmd.Process.Signal(signal))

			assert.Equal(t, 0, s.waitExit(t, 15*time.Second))
			assert.Contains(t, s.output.String(), "Server stopped")
		})
	}
}

// GIVEN a request whose body is still being sent / WHEN the server receives SIGTERM
// THEN it waits for that request, answers it, then exits with code 0
func TestShutdown_LetsARequestInProgressFinish(t *testing.T) {
	s := startServer(t)
	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", s.addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(30*time.Second)))
	chunk := func(data string) string { return fmt.Sprintf("%x\r\n%s\r\n", len(data), data) }

	reply := bufio.NewReader(conn)
	_, err = fmt.Fprintf(conn, "POST /api/v1/users HTTP/1.1\r\nHost: %s\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\nExpect: 100-continue\r\n\r\n",
		s.addr)
	require.NoError(t, err)
	// net/http sends "100 Continue" only when the handler first reads the body: from then on the
	// request is in progress, which is the moment to send part of the body and the signal.
	interim, err := http.ReadResponse(reply, nil)
	require.NoError(t, err)
	require.NoError(t, interim.Body.Close())
	require.Equal(t, http.StatusContinue, interim.StatusCode)
	_, err = fmt.Fprint(conn, chunk(`{"name":"Slow",`))
	require.NoError(t, err)

	require.NoError(t, s.cmd.Process.Signal(syscall.SIGTERM))
	// Showing that the server does not exit needs a bounded wait: an exit that ignored the request
	// would come at once, far within these 500 ms; the real exit is checked at the end.
	select {
	case err := <-s.exited:
		t.Fatalf("the server exited during the request: %v\n%s", err, s.output.String())
	case <-time.After(500 * time.Millisecond):
	}
	_, err = fmt.Fprint(conn, chunk(`"email":"slow@x.io"}`)+"0\r\n\r\n")
	require.NoError(t, err)

	resp, err := http.ReadResponse(reply, nil)
	require.NoError(t, err, "the request in progress must be answered")
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, 0, s.waitExit(t, 15*time.Second))
	assert.Contains(t, s.output.String(), "Server stopped")
}

// GIVEN CORS_ALLOWED_ORIGINS / WHEN the real binary answers / THEN only the listed origin is allowed
func TestStartup_CORSAllowedOriginsFromEnvironment(t *testing.T) {
	s := startServer(t, "CORS_ALLOWED_ORIGINS=https://app.example, https://admin.example")

	for origin, want := range map[string]string{
		"https://admin.example": "https://admin.example",
		"https://evil.example":  "",
	} {
		t.Run(origin, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+s.addr+"/health", nil)
			require.NoError(t, err)
			req.Header.Set("Origin", origin)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			_ = resp.Body.Close()

			assert.Equal(t, want, resp.Header.Get("Access-Control-Allow-Origin"), origin)
			assert.NotEmpty(t, resp.Header.Get("X-Request-Id"))
		})
	}
	assert.False(t, strings.Contains(s.output.String(), "panic"))
}
