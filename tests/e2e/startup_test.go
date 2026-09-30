package e2e

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildServer compiles the real command (main.go) once per test.
func buildServer(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, "go.mod"))

	bin := filepath.Join(t.TempDir(), "server")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".") //nolint:gosec // G204: builds this repository's own command into the test temp dir
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)
	return bin
}

// serverEnv is the current environment without the server settings, plus the given ones.
func serverEnv(settings ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "DATABASE_URL=") && !strings.HasPrefix(kv, "PORT=") {
			env = append(env, kv)
		}
	}
	return append(env, settings...)
}

// runUntilExit starts the server and waits for it to stop; it must stop on its own.
func runUntilExit(t *testing.T, bin string, env []string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin) //nolint:gosec // G204: runs the binary built by buildServer
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "the server was expected to stop by itself")
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), string(out)
	}
	require.NoError(t, err)
	return 0, string(out)
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
	require.NoError(t, l.Close())
	return port
}

// GIVEN no DATABASE_URL / WHEN the server starts / THEN it exits with code 1 and says why
func TestStartup_WithoutDatabaseURL_ExitsWithCode1(t *testing.T) {
	code, out := runUntilExit(t, buildServer(t), serverEnv())

	assert.Equal(t, 1, code)
	assert.Contains(t, out, "DATABASE_URL not set")
}

// GIVEN an unreachable database / WHEN the server starts / THEN it exits with code 1
func TestStartup_UnreachableDatabase_ExitsWithCode1(t *testing.T) {
	env := serverEnv("DATABASE_URL=postgres://testuser:testpass@127.0.0.1:1/testdb?sslmode=disable&connect_timeout=2")

	code, out := runUntilExit(t, buildServer(t), env)

	assert.Equal(t, 1, code)
	assert.Contains(t, out, "failed to initialize database")
}

// GIVEN the port is already taken / WHEN the server starts / THEN it exits with code 1
func TestStartup_PortInUse_ExitsWithCode1(t *testing.T) {
	pg.Require(t)
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", ":0") //nolint:gosec // G102: the server binds the wildcard address, so the port must be taken there
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	port := fmt.Sprint(l.Addr().(*net.TCPAddr).Port)

	code, out := runUntilExit(t, buildServer(t), serverEnv("DATABASE_URL="+pg.DSN, "PORT="+port))

	assert.Equal(t, 1, code)
	assert.Contains(t, out, "address already in use")
}

// GIVEN a reachable database / WHEN the real command starts / THEN it serves /health on PORT
func TestStartup_ServesHealthOnConfiguredPort(t *testing.T) {
	pg.Require(t)
	port := freePort(t)
	cmd := exec.CommandContext(t.Context(), buildServer(t)) //nolint:gosec // G204: runs the binary built by buildServer
	cmd.Env = serverEnv("DATABASE_URL="+pg.DSN, "PORT="+port)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	url := "http://127.0.0.1:" + port + "/health"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	var status int
	require.Eventually(t, func() bool {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		status = resp.StatusCode
		return true
	}, 20*time.Second, 100*time.Millisecond, "the server never answered on %s", url)
	assert.Equal(t, http.StatusOK, status)
}
