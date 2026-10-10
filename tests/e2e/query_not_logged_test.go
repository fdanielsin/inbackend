package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// queryMarker is recognizable in a log line, raw ("Marker-...") or percent-encoded ("%4Darker-..."); the check
// looks for its tail, which both forms share.
const (
	queryMarker = "Marker-7b3e91"
	markerTail  = "7b3e91"
)

type loggedRequest struct {
	method string
	path   string
	body   string
}

func get(path string) loggedRequest { return loggedRequest{method: http.MethodGet, path: path} }

// requireNeverLogged sends the requests and checks that each left exactly one access line, and that the secret
// is in no log line and in no answer.
func requireNeverLogged(t *testing.T, s *testServer, logs *syncBuffer, requests []loggedRequest) {
	t.Helper()
	before := len(logLines(t, logs, "Request"))
	for _, req := range requests {
		r := s.doRaw(req.method, req.path, req.body)
		assert.NotContains(t, string(r.body), markerTail, "%s %s: the answer must not echo the query", req.method, req.path)
	}
	assert.Len(t, logLines(t, logs, "Request"), before+len(requests), "each request leaves one access line, so the check sees them all")
	assert.NotContains(t, logs.String(), markerTail, "no log line may carry the query")
}

// GIVEN a request whose query carries a recognizable value / WHEN it is answered (success, refusal, unknown route)
// THEN no log line carries that value, in any family of routes
func TestRequestLogs_NeverCarryTheQuery(t *testing.T) {
	logs := captureLogs(t)
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	s.createTask(owner.ID, "Buy milk", "")
	tasks := fmt.Sprintf("/api/v1/users/%d/tasks", owner.ID)
	q := "marker=" + queryMarker

	t.Run("SEC-R08: reads with a query leave no trace of it (happy path)", func(t *testing.T) {
		requireNeverLogged(t, s, logs, []loggedRequest{
			get("/api/v1/users?" + q),
			get(fmt.Sprintf("/api/v1/users/%d?%s", owner.ID, q)),
			get(tasks + "?completed=true&" + q),
			get(tasks + "?" + q + "&completed=false"),
			get("/api/v1/stats?" + q),
			get("/health?" + q),
			get("/ready?" + q),
			{method: http.MethodHead, path: "/api/v1/users?" + q},
		})
		assert.NotEmpty(t, logLines(t, logs, "Listing user tasks"), "the control line is logged, so the check can see it")
	})

	t.Run("SEC-R08: writes with a query leave no trace of it (happy path)", func(t *testing.T) {
		requireNeverLogged(t, s, logs, []loggedRequest{
			{method: http.MethodPost, path: "/api/v1/users?" + q, body: `{"name":"Ada","email":"ada@example.com"}`},
			{method: http.MethodPost, path: "/api/v1/tasks?" + q, body: fmt.Sprintf(`{"user_id":%d,"title":"New"}`, owner.ID)},
			{method: http.MethodPut, path: "/api/v1/tasks/1?" + q, body: `{"completed":true}`},
			{method: http.MethodDelete, path: "/api/v1/tasks/1?" + q},
		})
	})

	t.Run("SEC-R08: refusals with a query leave no trace of it (sad path)", func(t *testing.T) {
		requireNeverLogged(t, s, logs, []loggedRequest{
			get("/api/v1/users/abc?" + q),
			get(tasks + "?completed=maybe&" + q),
			get(tasks + "?completed=true;" + q),
			get(tasks + "?completed=%zz&" + q),
			get(tasks + "?completed=true&completed=false&" + q),
			get("/api/v1/users/999?" + q),
			get("/nope?" + q),
			{method: http.MethodPost, path: "/health?" + q},
			{method: http.MethodPut, path: "/api/v1/tasks/abc?" + q, body: `{"completed":true}`},
			{method: http.MethodPost, path: "/api/v1/users?" + q, body: `{"name":`},
			{method: http.MethodPost, path: "/api/v1/users?" + q, body: `{"name":"Ada","email":"ada@example.com"}`},
		})
		assert.NotEmpty(t, logLines(t, logs, "Invalid completed filter"), "the refusals are logged")
	})

	t.Run("SEC-R08: odd queries leave no trace of them (limit)", func(t *testing.T) {
		requireNeverLogged(t, s, logs, []loggedRequest{
			get("/api/v1/users?%4Darker-7b3e91=1"),
			get("/api/v1/users?x=%4Darker-7b3e91"),
			get("/api/v1/users?x=" + strings.Repeat("a", 4000) + queryMarker),
			get("/api/v1/users?" + strings.Repeat("k=v&", 500) + q),
			get(tasks + "?completed=%74rue&" + q),
			get(tasks + "?;" + q),
			get(tasks + "?completed&" + q),
			get("/nope/" + strings.Repeat("a", 2000) + "?" + q),
		})
	})

	t.Run("SEC-R08: a failing request leaves no trace of its query (exception)", func(t *testing.T) {
		s.closeDatabase()

		requireNeverLogged(t, s, logs, []loggedRequest{
			get(tasks + "?completed=true&" + q),
			get("/api/v1/users?" + q),
			get("/api/v1/stats?" + q),
			get("/ready?" + q),
		})
		assert.NotEmpty(t, logLines(t, logs, "Request failed"), "the failures are logged")
	})
}
