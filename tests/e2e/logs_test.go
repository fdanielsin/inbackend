package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Personal data sent by the tests below; none of it may reach a log (B-08).
const (
	personName  = "Zoé Secretname"
	personEmail = "zoe.secret@example.org"
)

// forbiddenInLogs lists what must never be logged: the name, the email as sent
// and as stored, and the raw request body.
func forbiddenInLogs(body string) []string {
	return []string{personName, "Secretname", personEmail, strings.ToUpper(personEmail), body}
}

// syncBuffer collects the output of a running process safely.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func postJSON(t *testing.T, url, body string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

// GIVEN the real binary / WHEN a user is created, created again and sent with an invalid body
// THEN neither the name, nor the email, nor the request body appears anywhere in stdout or stderr (B-08)
func TestServerOutput_NeverContainsPersonalData(t *testing.T) {
	pg.Require(t)
	port := freePort(t)
	output := &syncBuffer{}
	cmd := exec.CommandContext(t.Context(), buildServer(t)) //nolint:gosec // G204: runs the binary built by buildServer
	cmd.Env = serverEnv("DATABASE_URL="+pg.DSN, "PORT="+port)
	cmd.Stdout = output
	cmd.Stderr = output
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	base := "http://127.0.0.1:" + port
	require.Eventually(t, func() bool {
		resp, err := http.Get(base + "/health") //nolint:noctx // polling a local test server
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 20*time.Second, 100*time.Millisecond)
	body, err := json.Marshal(map[string]string{"name": personName, "email": strings.ToUpper(personEmail)})
	require.NoError(t, err)
	invalid := `{"name":"` + personName + `","email":1}`

	created := postJSON(t, base+"/api/v1/users", string(body))
	duplicate := postJSON(t, base+"/api/v1/users", string(body))
	rejected := postJSON(t, base+"/api/v1/users", invalid)

	assert.Equal(t, []int{http.StatusCreated, http.StatusConflict, http.StatusBadRequest}, []int{created, duplicate, rejected})
	require.Eventually(t, func() bool { return strings.Count(output.String(), "/api/v1/users") >= 3 }, 5*time.Second, 50*time.Millisecond,
		"the request logs must be written before they are checked:\n%s", output.String())
	logs := output.String()
	for _, personal := range append(forbiddenInLogs(string(body)), invalid) {
		assert.NotContains(t, logs, personal)
	}
}

// GIVEN the database is down / WHEN a user is created / THEN the 500 is logged without the name, the email or the body (B-08)
func TestServerLogs_InternalErrorWithoutPersonalData(t *testing.T) {
	var logs syncBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	s := newTestServer(t)
	s.closeDatabase()
	body, err := json.Marshal(map[string]string{"name": personName, "email": personEmail})
	require.NoError(t, err)

	r := s.do(http.MethodPost, "/api/v1/users", string(body))

	require.NotEmpty(t, logs.String(), "the failure must be logged")
	for _, personal := range forbiddenInLogs(string(body)) {
		assert.NotContains(t, logs.String(), personal)
	}
	requireError(t, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
}
