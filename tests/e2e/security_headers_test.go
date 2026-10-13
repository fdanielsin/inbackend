package e2e

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nosniffCase struct {
	name     string // no "/" in a name: Go reads it as a sub-test level
	category string // happy path, sad path, limit, boundary or exception
	method   string
	path     string
	body     string
	chunked  bool // sent without Content-Length
	headers  map[string]string
	status   int
}

// send performs the case on a real HTTP server, so that net/http applies Content-Length, chunking and HEAD.
func (tc nosniffCase) send(t *testing.T, base string) exchange {
	t.Helper()
	var body io.Reader
	if tc.body != "" {
		body = strings.NewReader(tc.body)
		if tc.chunked {
			body = io.NopCloser(body) // an opaque reader: the length is unknown, the client chunks the body
		}
	}
	req, err := http.NewRequestWithContext(context.Background(), tc.method, base+tc.path, body)
	require.NoError(t, err)
	if tc.body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	for name, value := range tc.headers {
		req.Header.Set(name, value)
	}
	resp, err := callClient.Do(req)
	require.NoError(t, err)
	read, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return exchange{status: resp.StatusCode, header: resp.Header, body: read, contentLength: resp.ContentLength}
}

func runNosniffCases(t *testing.T, base string, cases []nosniffCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run("SEC-R04: "+tc.name+" ("+tc.category+")", func(t *testing.T) {
			got := tc.send(t, base)

			require.Equal(t, tc.status, got.status, "the case must reach the answer it is meant to test; body: %s", got.body)
			assert.Equal(t, []string{"nosniff"}, got.header.Values("X-Content-Type-Options"))
		})
	}
}

const preflightOrigin = "https://app.example"

func preflight(origin, method string) map[string]string {
	return map[string]string{echo.HeaderOrigin: origin, echo.HeaderAccessControlRequestMethod: method}
}

// GIVEN every kind of answer the API gives (success, client error, preflight, size limit, unknown route)
// WHEN each is requested on a real server / THEN each carries X-Content-Type-Options: nosniff, exactly once
func TestSecurityHeaders_NosniffOnEveryAnswer(t *testing.T) {
	s := seededServer(t, app.WithAPIDocs(contractFile(t)), app.WithAllowedOrigins([]string{preflightOrigin}))
	s.createTask(1, "To delete", "") // task 2
	s.e.GET("/panic", func(*echo.Context) error { panic("boom") })
	base := liveServer(t, s)
	longPath := "/" + strings.Repeat("x", 5000)
	jsonUser := `{"name":"Ada","email":"ada@example.com"}`
	oversizedJSON := `{"name":"` + strings.Repeat("a", maxBody) + `","email":"a@b.co"}`

	runNosniffCases(t, base, []nosniffCase{
		{name: "health probe", category: "happy path", method: http.MethodGet, path: "/health", status: http.StatusOK},
		{name: "readiness probe", category: "happy path", method: http.MethodGet, path: "/ready", status: http.StatusOK},
		{name: "user list", category: "happy path", method: http.MethodGet, path: "/api/v1/users", status: http.StatusOK},
		{name: "one user", category: "happy path", method: http.MethodGet, path: "/api/v1/users/1", status: http.StatusOK},
		{name: "task list with the filter", category: "happy path", method: http.MethodGet, path: "/api/v1/users/1/tasks?completed=false", status: http.StatusOK},
		{name: "statistics", category: "happy path", method: http.MethodGet, path: "/api/v1/stats", status: http.StatusOK},
		{name: "user creation", category: "happy path", method: http.MethodPost, path: "/api/v1/users", body: jsonUser, status: http.StatusCreated},
		{name: "task creation", category: "happy path", method: http.MethodPost, path: "/api/v1/tasks", body: `{"user_id":1,"title":"New"}`, status: http.StatusCreated},
		{name: "task update", category: "happy path", method: http.MethodPut, path: "/api/v1/tasks/1", body: `{"completed":true}`, status: http.StatusOK},
		{name: "task deletion", category: "happy path", method: http.MethodDelete, path: "/api/v1/tasks/2", status: http.StatusOK},
		{name: "user rename", category: "happy path", method: http.MethodPut, path: "/api/v1/users/1", body: `{"name":"Ada Lovelace"}`, status: http.StatusOK},
		{name: "user deletion", category: "happy path", method: http.MethodDelete, path: "/api/v1/users/2", status: http.StatusOK},
		{name: "HEAD of a list", category: "happy path", method: http.MethodHead, path: "/api/v1/users", status: http.StatusOK},
		{name: "Swagger page", category: "happy path", method: http.MethodGet, path: "/docs", status: http.StatusOK},
		{name: "contract file", category: "happy path", method: http.MethodGet, path: "/openapi.yaml", status: http.StatusOK},

		{name: "invalid user id", category: "sad path", method: http.MethodGet, path: "/api/v1/users/abc", status: http.StatusBadRequest},
		{name: "invalid task id on update", category: "sad path", method: http.MethodPut, path: "/api/v1/tasks/abc", body: `{"completed":true}`, status: http.StatusBadRequest},
		{name: "invalid user id on rename", category: "sad path", method: http.MethodPut, path: "/api/v1/users/abc", body: `{"name":"Ada"}`, status: http.StatusBadRequest},
		{name: "rename with an e-mail", category: "sad path", method: http.MethodPut, path: "/api/v1/users/1", body: `{"name":"Ada","email":"a@b.co"}`, status: http.StatusBadRequest},
		{name: "unknown user on rename", category: "sad path", method: http.MethodPut, path: "/api/v1/users/999", body: `{"name":"Ada"}`, status: http.StatusNotFound},
		{name: "unknown user on deletion", category: "sad path", method: http.MethodDelete, path: "/api/v1/users/999", status: http.StatusNotFound},
		{name: "malformed JSON body", category: "sad path", method: http.MethodPost, path: "/api/v1/users", body: `{"name":`, status: http.StatusBadRequest},
		{name: "invalid e-mail", category: "sad path", method: http.MethodPost, path: "/api/v1/users", body: `{"name":"Ada","email":"ada"}`, status: http.StatusBadRequest},
		{name: "malformed completed filter", category: "sad path", method: http.MethodGet, path: "/api/v1/users/1/tasks?completed=true;", status: http.StatusBadRequest},
		{name: "unknown user", category: "sad path", method: http.MethodGet, path: "/api/v1/users/999", status: http.StatusNotFound},
		{name: "unknown task", category: "sad path", method: http.MethodDelete, path: "/api/v1/tasks/999", status: http.StatusNotFound},
		{name: "unknown route", category: "sad path", method: http.MethodGet, path: "/nope", status: http.StatusNotFound},
		{name: "HEAD on an unknown route", category: "sad path", method: http.MethodHead, path: "/nope", status: http.StatusNotFound},
		{name: "method not allowed", category: "sad path", method: http.MethodPost, path: "/health", status: http.StatusMethodNotAllowed},
		{name: "e-mail already taken", category: "sad path", method: http.MethodPost, path: "/api/v1/users", body: `{"name":"Owner","email":"owner@example.com"}`, status: http.StatusConflict},

		{name: "preflight of a listed origin", category: "limit", method: http.MethodOptions, path: "/api/v1/tasks/1", headers: preflight(preflightOrigin, http.MethodPut), status: http.StatusNoContent},
		{name: "preflight of a user deletion", category: "limit", method: http.MethodOptions, path: "/api/v1/users/1", headers: preflight(preflightOrigin, http.MethodDelete), status: http.StatusNoContent},
		{name: "preflight of an unlisted origin", category: "limit", method: http.MethodOptions, path: "/api/v1/tasks/1", headers: preflight("https://evil.example", http.MethodPut), status: http.StatusNoContent},
		{name: "preflight asking for a method the API does not have", category: "limit", method: http.MethodOptions, path: "/api/v1/tasks/1", headers: preflight(preflightOrigin, http.MethodPatch), status: http.StatusNoContent},
		{name: "preflight on an unknown route", category: "limit", method: http.MethodOptions, path: "/nope", headers: preflight(preflightOrigin, http.MethodGet), status: http.StatusNoContent},
		{name: "cross-origin request from an unlisted origin", category: "limit", method: http.MethodGet, path: "/health", headers: map[string]string{echo.HeaderOrigin: "https://evil.example"}, status: http.StatusOK},
		{name: "very long unknown path", category: "limit", method: http.MethodGet, path: longPath, status: http.StatusNotFound},
		{name: "non-ASCII unknown path", category: "limit", method: http.MethodGet, path: "/%F0%9F%9A%80/%C3%BCn%C3%AF", status: http.StatusNotFound},

		{name: "body of exactly 1 MiB", category: "boundary", method: http.MethodPost, path: "/api/v1/users", body: strings.Repeat("a", maxBody), status: http.StatusBadRequest},
		{name: "body of 1 MiB plus one byte", category: "boundary", method: http.MethodPost, path: "/api/v1/users", body: strings.Repeat("a", maxBody+1), status: http.StatusRequestEntityTooLarge},
		{name: "oversized body without Content-Length", category: "boundary", method: http.MethodPost, path: "/api/v1/users", chunked: true, body: oversizedJSON, status: http.StatusRequestEntityTooLarge},
		{name: "largest user id", category: "boundary", method: http.MethodGet, path: "/api/v1/users/4294967295", status: http.StatusNotFound},
		{name: "user id one above the largest", category: "boundary", method: http.MethodGet, path: "/api/v1/users/4294967296", status: http.StatusBadRequest},

		{name: "recovered panic", category: "exception", method: http.MethodGet, path: "/panic", status: http.StatusInternalServerError},
	})
}

// GIVEN the database is down / WHEN reading, writing and probing readiness / THEN the 500 and the 503 carry nosniff too
func TestSecurityHeaders_NosniffWhenTheDatabaseIsDown(t *testing.T) {
	s := newTestServer(t)
	s.closeDatabase()
	base := liveServer(t, s)

	runNosniffCases(t, base, []nosniffCase{
		{name: "user list with the database down", category: "exception", method: http.MethodGet, path: "/api/v1/users", status: http.StatusInternalServerError},
		{name: "task update with the database down", category: "exception", method: http.MethodPut, path: "/api/v1/tasks/1", body: `{"completed":true}`, status: http.StatusInternalServerError},
		{name: "user rename with the database down", category: "exception", method: http.MethodPut, path: "/api/v1/users/1", body: `{"name":"Ada"}`, status: http.StatusInternalServerError},
		{name: "user deletion with the database down", category: "exception", method: http.MethodDelete, path: "/api/v1/users/1", status: http.StatusInternalServerError},
		{name: "readiness probe with the database down", category: "exception", method: http.MethodGet, path: "/ready", status: http.StatusServiceUnavailable},
		{name: "HEAD of the readiness probe with the database down", category: "exception", method: http.MethodHead, path: "/ready", status: http.StatusServiceUnavailable},
	})
}
