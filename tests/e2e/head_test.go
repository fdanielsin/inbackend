package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exchange is a response read from a real HTTP server, so that net/http applies its HEAD semantics.
type exchange struct {
	status           int
	header           http.Header
	body             []byte
	contentLength    int64
	transferEncoding []string
}

// liveServer serves the router of s over a real socket for the duration of the test.
func liveServer(t *testing.T, s *testServer) string {
	t.Helper()
	server := httptest.NewServer(s.e)
	t.Cleanup(server.Close)
	return server.URL
}

func call(t *testing.T, method, url, body string, headers map[string]string) exchange {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, url, strings.NewReader(body))
	require.NoError(t, err)
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	read, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return exchange{status: resp.StatusCode, header: resp.Header, body: read, contentLength: resp.ContentLength, transferEncoding: resp.TransferEncoding}
}

// callOperation is call for a request that is an operation of the contract: the response is
// validated against openapi.yaml and counts for its coverage (QP-02).
func callOperation(t *testing.T, method, url, body string, headers map[string]string) exchange {
	t.Helper()
	got := call(t, method, url, body, headers)
	req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
	require.NoError(t, err)
	contract.ValidateResponse(t, req, got.status, got.header, got.body)
	return got
}

// asResponse lets the helpers written for the router in memory check an exchange.
func (e exchange) asResponse() response {
	return response{code: e.status, body: e.body, header: e.header}
}

// requireHeadLikeGet checks RFC 9110 §9.3.2: same status and header fields as GET, no content.
// Date and X-Request-Id change on every response; X-Request-Id must still be present.
func requireHeadLikeGet(t *testing.T, get, head exchange, what string) {
	t.Helper()
	require.Equal(t, get.status, head.status, what)
	assert.Empty(t, head.body, "%s: HEAD has no content", what)
	assert.NotEmpty(t, head.header.Get(echo.HeaderXRequestID), what)
	names := func(h http.Header) []string {
		var keys []string
		for key := range h {
			if key != "Date" && key != echo.HeaderXRequestID {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		return keys
	}
	assert.Equal(t, names(get.header), names(head.header), "%s: same header fields", what)
	for _, key := range names(get.header) {
		assert.Equal(t, get.header.Values(key), head.header.Values(key), "%s: %s", what, key)
	}
	assert.Equal(t, get.contentLength, head.contentLength, "%s: same Content-Length (-1 = not sent)", what)
	assert.Equal(t, get.transferEncoding, head.transferEncoding, "%s: same framing", what)
}

// seededServer has user 1 with one task, so that the GET routes answer 200.
func seededServer(t *testing.T, opts ...app.Option) *testServer {
	t.Helper()
	s := newTestServer(t, opts...)
	owner := s.createUser("Owner", "owner@example.com")
	s.createTask(owner.ID, "Buy milk", "")
	return s
}

// GIVEN every GET route (the docs ones with the option) and their error answers, in English and in French
// WHEN HEAD is sent to a real server / THEN the same status and header fields as GET, Content-Length included, and no body
func TestHead_AnswersLikeGetWithoutContent(t *testing.T) {
	base := liveServer(t, seededServer(t, app.WithAPIDocs(contractFile(t))))
	operations := []string{
		"/health", "/ready", "/api/v1/users", "/api/v1/users/1", "/api/v1/users/1/tasks", "/api/v1/stats",
		"/docs", "/openapi.yaml",
		"/api/v1/users/abc", "/api/v1/users/abc/tasks", "/api/v1/users/999", "/api/v1/users/999/tasks",
		"/api/v1/users?page=2",
	}
	outsideOperations := []string{"/api/v1/unknown", "/docs/"}

	for _, language := range []string{"en", "fr"} {
		headers := map[string]string{headerAcceptLanguage: language}
		for _, path := range operations {
			t.Run(path+" "+language, func(t *testing.T) {
				get := callOperation(t, http.MethodGet, base+path, "", headers)
				head := callOperation(t, http.MethodHead, base+path, "", headers)

				requireHeadLikeGet(t, get, head, "HEAD "+path)
			})
		}
		for _, path := range outsideOperations {
			t.Run(path+" "+language, func(t *testing.T) {
				get := call(t, http.MethodGet, base+path, "", headers)
				contract.ValidateErrorEnvelope(t, get.body)
				head := call(t, http.MethodHead, base+path, "", headers)

				requireHeadLikeGet(t, get, head, "HEAD "+path)
			})
		}
	}
}

// GIVEN the docs option / WHEN GET and HEAD /openapi.yaml on a real server
// THEN both announce the size of the contract in Content-Length, without chunked framing (decided)
func TestHead_OpenAPIAnnouncesTheContractSize(t *testing.T) {
	base := liveServer(t, newDocsServer(t))
	size := int64(len(contractFile(t)))

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		r := callOperation(t, method, base+"/openapi.yaml", "", nil)

		require.Equal(t, http.StatusOK, r.status, method)
		assert.Equal(t, size, r.contentLength, method)
		assert.Empty(t, r.transferEncoding, method)
	}
}

// GIVEN the database is down / WHEN HEAD on each reading route and on /ready / THEN 500 and 503 like GET, without body
func TestHead_ServerErrorsLikeGet(t *testing.T) {
	s := newTestServer(t)
	base := liveServer(t, s)
	s.closeDatabase()

	for path, status := range map[string]int{
		"/api/v1/users":         http.StatusInternalServerError,
		"/api/v1/users/1":       http.StatusInternalServerError,
		"/api/v1/users/1/tasks": http.StatusInternalServerError,
		"/api/v1/stats":         http.StatusInternalServerError,
		"/ready":                http.StatusServiceUnavailable,
	} {
		get := callOperation(t, http.MethodGet, base+path, "", nil)
		head := callOperation(t, http.MethodHead, base+path, "", nil)

		assert.Equal(t, status, get.status, path)
		requireHeadLikeGet(t, get, head, "HEAD "+path)
	}
}

// GIVEN a HEAD request carrying a body / THEN it is treated like the same GET: a small body is ignored,
// a body over 1 MiB gets the 413 that GET gets
func TestHead_WithABodyLikeGet(t *testing.T) {
	base := liveServer(t, seededServer(t))

	get := callOperation(t, http.MethodGet, base+"/api/v1/users", `{"x":1}`, nil)
	head := callOperation(t, http.MethodHead, base+"/api/v1/users", `{"x":1}`, nil)
	require.Equal(t, http.StatusOK, head.status)
	requireHeadLikeGet(t, get, head, "HEAD with a small body")

	// The 413 is the global rule of the contract (info), declared on no operation.
	tooLarge := strings.Repeat("a", 1<<20+1)
	get = call(t, http.MethodGet, base+"/api/v1/users", tooLarge, nil)
	contract.ValidateErrorEnvelope(t, get.body)
	head = call(t, http.MethodHead, base+"/api/v1/users", tooLarge, nil)
	require.Equal(t, http.StatusRequestEntityTooLarge, head.status)
	requireHeadLikeGet(t, get, head, "HEAD with a body over 1 MiB")
}

// allowed returns the methods of the Allow header.
func allowed(e exchange) []string {
	var methods []string
	for _, method := range strings.Split(e.header.Get(echo.HeaderAllow), ",") {
		if method = strings.TrimSpace(method); method != "" {
			methods = append(methods, method)
		}
	}
	return methods
}

// GIVEN a method a GET route does not handle / THEN 405 with Allow listing GET and HEAD;
// GIVEN HEAD on a route without GET / THEN 405 as before, Allow without GET nor HEAD, no body
func TestHead_AllowHeader(t *testing.T) {
	base := liveServer(t, seededServer(t))

	for _, rq := range []struct{ method, path string }{
		{http.MethodDelete, "/health"}, {http.MethodPost, "/api/v1/users/1"}, {http.MethodPut, "/api/v1/stats"},
	} {
		r := call(t, rq.method, base+rq.path, "", nil)
		require.Equal(t, http.StatusMethodNotAllowed, r.status, rq.method+" "+rq.path)
		contract.ValidateErrorEnvelope(t, r.body)
		assert.Contains(t, allowed(r), http.MethodGet, rq.path)
		assert.Contains(t, allowed(r), http.MethodHead, rq.path)
	}

	for path, want := range map[string][]string{
		"/api/v1/tasks":   {http.MethodPost},
		"/api/v1/tasks/1": {http.MethodPut, http.MethodDelete},
	} {
		r := call(t, http.MethodHead, base+path, "", nil)
		require.Equal(t, http.StatusMethodNotAllowed, r.status, "HEAD "+path)
		assert.Empty(t, r.body)
		for _, method := range want {
			assert.Contains(t, allowed(r), method, path)
		}
		assert.NotContains(t, allowed(r), http.MethodGet, path)
		assert.NotContains(t, allowed(r), http.MethodHead, path)
	}
}

// GIVEN CORS / WHEN a simple HEAD with an Origin, and a preflight asking for HEAD
// THEN the same CORS headers as GET, and HEAD allowed by the preflight (decided)
func TestHead_CORS(t *testing.T) {
	base := liveServer(t, seededServer(t))
	origin := map[string]string{echo.HeaderOrigin: "https://app.example.com"}

	get := callOperation(t, http.MethodGet, base+"/api/v1/users", "", origin)
	head := callOperation(t, http.MethodHead, base+"/api/v1/users", "", origin)
	requireHeadLikeGet(t, get, head, "HEAD with Origin")
	assert.Equal(t, "*", head.header.Get(echo.HeaderAccessControlAllowOrigin))

	preflight := call(t, http.MethodOptions, base+"/api/v1/users", "", map[string]string{
		echo.HeaderOrigin: "https://app.example.com", echo.HeaderAccessControlRequestMethod: http.MethodHead,
	})
	assert.Equal(t, http.StatusNoContent, preflight.status)
	assert.Contains(t, strings.Split(strings.ReplaceAll(preflight.header.Get(echo.HeaderAccessControlAllowMethods), " ", ""), ","), http.MethodHead)
}

// GIVEN HEAD requests / THEN one access line each, method HEAD, the route and status of GET;
// a successful probe stays at DEBUG and an error at the level of the same GET
func TestHead_AccessLog(t *testing.T) {
	logs := captureLogs(t)
	base := liveServer(t, seededServer(t))

	callOperation(t, http.MethodHead, base+"/health", "", nil)
	callOperation(t, http.MethodHead, base+"/api/v1/users/abc", "", nil)
	callOperation(t, http.MethodHead, base+"/api/v1/users", "", nil)

	var got []string
	for _, line := range logLines(t, logs, "Request") {
		if line["method"] == http.MethodHead {
			got = append(got, fmt.Sprintf("%v %v %v", line["route"], line["level"], line["status"]))
		}
	}
	assert.Equal(t, []string{"/health DEBUG 200", "/api/v1/users/:id WARN 400", "/api/v1/users INFO 200"}, got)
}

// GIVEN the real binary (docs on by default outside production) / WHEN HEAD on probes and docs
// THEN 200, no content, the header fields of GET
func TestHead_RealBinary(t *testing.T) {
	server := startServer(t)
	base := "http://" + server.addr
	for _, path := range []string{"/health", "/ready", "/docs", "/openapi.yaml"} {
		get := callOperation(t, http.MethodGet, base+path, "", nil)
		head := callOperation(t, http.MethodHead, base+path, "", nil)

		require.Equal(t, http.StatusOK, head.status, path)
		requireHeadLikeGet(t, get, head, "binary HEAD "+path)
	}
}
