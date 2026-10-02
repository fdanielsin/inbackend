package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs routes slog to a JSON buffer; call it before newTestServer, since
// the router takes its logger when it is built.
func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	logs := &syncBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return logs
}

// logLines returns the JSON log records with the given message.
func logLines(t *testing.T, logs *syncBuffer, msg string) []map[string]any {
	t.Helper()
	var lines []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(logs.String()))
	scanner.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for scanner.Scan() {
		var line map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &line), scanner.Text())
		if line["msg"] == msg {
			lines = append(lines, line)
		}
	}
	return lines
}

// sendWithRequestID sends a request carrying the given X-Request-Id ("" = none).
func (s *testServer) sendWithRequestID(method, path, requestID string) *httptest.ResponseRecorder {
	s.t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if requestID != "" {
		req.Header.Set(echo.HeaderXRequestID, requestID)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

func requireGeneratedID(t *testing.T, id string) {
	t.Helper()
	_, err := uuid.Parse(id)
	require.NoError(t, err, "a UUID must be generated, got %q", id)
}

// GIVEN any response (success, 400, 404 of a handler, 404 of the router, 500) / THEN it carries an X-Request-Id
func TestRequestID_OnEveryResponse(t *testing.T) {
	s := newTestServer(t)
	s.e.GET("/panic", func(*echo.Context) error { panic("boom") })

	for _, path := range []string{"/health", "/api/v1/users/abc", "/api/v1/users/999", "/api/v1/unknown", "/panic"} {
		t.Run(path, func(t *testing.T) {
			requireGeneratedID(t, s.in(t).sendWithRequestID(http.MethodGet, path, "").Header().Get(echo.HeaderXRequestID))
		})
	}
}

// GIVEN a client id of 1 to 64 allowed characters / THEN it is echoed back unchanged
func TestRequestID_AcceptableClientIDIsKept(t *testing.T) {
	s := newTestServer(t)

	for _, id := range []string{"a", "abc-123.x_Y", strings.Repeat("z", 64)} {
		t.Run(fmt.Sprintf("%d characters %.12q", len(id), id), func(t *testing.T) {
			assert.Equal(t, id, s.in(t).sendWithRequestID(http.MethodGet, "/health", id).Header().Get(echo.HeaderXRequestID))
		})
	}
}

// GIVEN a client id that is too long or holds other characters / THEN it is replaced by a generated UUID
func TestRequestID_UnacceptableClientIDIsReplaced(t *testing.T) {
	s := newTestServer(t)

	for _, id := range []string{strings.Repeat("z", 65), "bad id", "a<script>", "é", "a\tb", "a/b"} {
		t.Run(fmt.Sprintf("%d characters %.12q", len(id), id), func(t *testing.T) {
			got := s.in(t).sendWithRequestID(http.MethodGet, "/health", id).Header().Get(echo.HeaderXRequestID)
			assert.NotEqual(t, id, got)
			requireGeneratedID(t, got)
		})
	}
}

// GIVEN a request / THEN one "Request" access log line with method, path without query, route, status,
// latency, remote ip and the same request_id as the response header and the controller logs
func TestAccessLog_OneLinePerRequest(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)

	rec := s.sendWithRequestID(http.MethodGet, "/api/v1/users/999?secret=hunter2", "trace-42")

	require.Equal(t, http.StatusNotFound, rec.Code)
	lines := logLines(t, logs, "Request")
	require.Len(t, lines, 1)
	line := lines[0]
	assert.Equal(t, "GET", line["method"])
	assert.Equal(t, "/api/v1/users/999", line["path"])
	assert.Equal(t, "/api/v1/users/:id", line["route"])
	assert.InDelta(t, 404, line["status"], 0)
	assert.Contains(t, line, "latency")
	assert.Contains(t, line, "remote_ip")
	assert.Equal(t, "trace-42", line["request_id"])
	assert.Equal(t, "WARN", line["level"])
	assert.NotContains(t, logs.String(), "hunter2", "the query string is never logged")
	controller := logLines(t, logs, "Error fetching user")
	require.NotEmpty(t, controller, "the controller logs the not-found")
	assert.Equal(t, "trace-42", controller[0]["request_id"])
	assert.Equal(t, "INFO", controller[0]["level"], "a not-found is expected traffic, logged as information")
}

// GIVEN a 409, a 500 of a controller, and errors produced by the router / THEN the error logs have the level of
// their status, and only server errors carry the error type (never the error text)
func TestErrorLogs_LevelAndTypeFollowStatus(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)
	s.createUser("Ada", "ada@x.io")
	s.e.GET("/http500", func(*echo.Context) error { return echo.NewHTTPError(http.StatusInternalServerError, "secret") })

	s.do(http.MethodPost, "/api/v1/users", `{"name":"Other","email":"ada@x.io"}`)
	s.doOutsideOperations(http.MethodGet, "/api/v1/unknown")
	r := s.doRaw(http.MethodGet, "/http500", "")
	requireError(t, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	s.closeDatabase()
	s.do(http.MethodGet, "/api/v1/users", "")

	conflict := logLines(t, logs, "Error creating user")
	require.Len(t, conflict, 1)
	assert.Equal(t, "WARN", conflict[0]["level"])
	assert.NotContains(t, conflict[0], "error_type")
	failed := logLines(t, logs, "Request failed")
	require.Len(t, failed, 2)
	assert.Equal(t, "WARN", failed[0]["level"], "unknown route")
	assert.NotContains(t, failed[0], "error_type")
	assert.Equal(t, "ERROR", failed[1]["level"], "HTTP 500")
	assert.Contains(t, failed[1], "error_type")
	down := logLines(t, logs, "Error listing users")
	require.Len(t, down, 1)
	assert.Equal(t, "ERROR", down[0]["level"])
	assert.Contains(t, down[0], "error_type")
	assert.NotContains(t, logs.String(), "secret")
}

// GIVEN a 2xx, a 4xx and a 5xx / THEN the access log level is INFO, WARN and ERROR
func TestAccessLog_LevelFollowsStatus(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)
	s.e.GET("/panic", func(*echo.Context) error { panic("boom") })

	for _, path := range []string{"/api/v1/stats", "/api/v1/users/abc", "/panic"} {
		s.sendWithRequestID(http.MethodGet, path, "")
	}

	var levels []any
	for _, line := range logLines(t, logs, "Request") {
		levels = append(levels, line["level"])
	}
	assert.Equal(t, []any{"INFO", "WARN", "ERROR"}, levels)
	assert.NotContains(t, logs.String(), "boom", "no error text in the logs")
}

// GIVEN the probes / WHEN they succeed / THEN their access log is DEBUG (hidden at INFO); a failed /ready is ERROR
func TestAccessLog_ProbesAreDebugUnlessTheyFail(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)

	s.sendWithRequestID(http.MethodGet, "/health", "")
	s.sendWithRequestID(http.MethodGet, "/ready", "")
	s.closeDatabase()
	s.sendWithRequestID(http.MethodGet, "/ready", "")

	var levels []string
	for _, line := range logLines(t, logs, "Request") {
		levels = append(levels, fmt.Sprintf("%v %v %v", line["route"], line["status"], line["level"]))
	}
	assert.Equal(t, []string{"/health 200 DEBUG", "/ready 200 DEBUG", "/ready 503 ERROR"}, levels)
}

func corsRequest(s *testServer, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/tasks/1", nil)
	req.Header.Set(echo.HeaderOrigin, origin)
	if method == http.MethodOptions {
		req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPut)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

// GIVEN a list of allowed origins / WHEN a listed origin asks / THEN it is allowed and X-Request-Id is exposed
func TestCORS_ListedOriginIsAllowed(t *testing.T) {
	s := newTestServer(t, app.WithAllowedOrigins([]string{"https://app.example", "https://admin.example"}))

	for _, origin := range []string{"https://app.example", "https://admin.example"} {
		t.Run(origin, func(t *testing.T) {
			preflight := corsRequest(s.in(t), http.MethodOptions, origin)
			assert.Equal(t, http.StatusNoContent, preflight.Code)
			assert.Equal(t, origin, preflight.Header().Get(echo.HeaderAccessControlAllowOrigin))

			simple := corsRequest(s.in(t), http.MethodGet, origin)
			assert.Equal(t, origin, simple.Header().Get(echo.HeaderAccessControlAllowOrigin))
			assert.Equal(t, echo.HeaderXRequestID, simple.Header().Get(echo.HeaderAccessControlExposeHeaders))
		})
	}
}

// GIVEN a list of allowed origins / WHEN another origin asks, even a near miss / THEN no CORS header is sent
func TestCORS_UnlistedOriginGetsNoHeader(t *testing.T) {
	s := newTestServer(t, app.WithAllowedOrigins([]string{"https://app.example"}))

	for _, origin := range []string{"https://evil.example", "https://app.example.evil", "http://app.example"} {
		for _, method := range []string{http.MethodOptions, http.MethodGet} {
			t.Run(method+" "+origin, func(t *testing.T) {
				assert.Empty(t, corsRequest(s.in(t), method, origin).Header().Get(echo.HeaderAccessControlAllowOrigin), "%s %s", method, origin)
			})
		}
	}
}
