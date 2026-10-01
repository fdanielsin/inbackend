package e2e

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonLogger(out *syncBuffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// GIVEN a router built with WithLogger / WHEN requests succeed, are invalid or fail on the database
// THEN every log line (controllers, access log, readiness) goes to that logger and none to slog's default
func TestWithLogger_ReceivesEveryLogLine(t *testing.T) {
	defaultLogs := captureLogs(t)
	injected := &syncBuffer{}
	s := newTestServer(t, app.WithLogger(jsonLogger(injected)))

	s.sendWithRequestID(http.MethodPost, "/api/v1/users", "")
	s.sendWithRequestID(http.MethodGet, "/api/v1/users/abc", "")
	s.sendWithRequestID(http.MethodGet, "/api/v1/unknown", "")
	s.closeDatabase()
	s.sendWithRequestID(http.MethodGet, "/ready", "")
	s.sendWithRequestID(http.MethodGet, "/api/v1/users", "")

	assert.Empty(t, defaultLogs.String(), "nothing may reach slog's default logger")
	assert.Len(t, logLines(t, injected, "Request"), 5, "one access line per request")
	assert.Len(t, logLines(t, injected, "Invalid user ID"), 1)
	assert.Len(t, logLines(t, injected, "Readiness check failed"), 1)
	assert.Len(t, logLines(t, injected, "Listing all users"), 1)
}

// GIVEN WithLogger(nil), alone or after another WithLogger / WHEN a request is served
// THEN nil is ignored: the logger in place (slog's default when NewRouter runs, or the one given before) is kept
func TestWithLogger_NilKeepsTheLoggerInPlace(t *testing.T) {
	t.Run("alone", func(t *testing.T) {
		defaultLogs := captureLogs(t)
		s := newTestServer(t, app.WithLogger(nil))

		s.sendWithRequestID(http.MethodGet, "/api/v1/users/abc", "")

		assert.Len(t, logLines(t, defaultLogs, "Request"), 1)
	})
	t.Run("after a logger", func(t *testing.T) {
		defaultLogs := captureLogs(t)
		injected := &syncBuffer{}
		s := newTestServer(t, app.WithLogger(jsonLogger(injected)), app.WithLogger(nil))

		s.sendWithRequestID(http.MethodGet, "/api/v1/users/abc", "")

		assert.Empty(t, defaultLogs.String())
		assert.Len(t, logLines(t, injected, "Request"), 1)
	})
}

// GIVEN a router built without WithLogger / WHEN slog's default changes afterwards
// THEN the router keeps the default it was built with (read once, at NewRouter)
func TestWithLogger_DefaultIsReadWhenTheRouterIsBuilt(t *testing.T) {
	atBuild := captureLogs(t)
	s := newTestServer(t)
	later := captureLogs(t)

	s.sendWithRequestID(http.MethodGet, "/api/v1/users/abc", "")

	require.Len(t, logLines(t, atBuild, "Request"), 1)
	assert.Empty(t, later.String())
}
