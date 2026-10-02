package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN the production router / THEN exactly the routes of the contract are registered, HEAD with every GET (QP-01)
func TestRouter_RegistersTheContractRoutes(t *testing.T) {
	s := newTestServer(t)

	var got []string
	for _, r := range s.e.Router().Routes() {
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)

	assert.Equal(t, []string{
		"DELETE /api/v1/tasks/:id",
		"GET /api/v1/stats",
		"GET /api/v1/users",
		"GET /api/v1/users/:id",
		"GET /api/v1/users/:id/tasks",
		"GET /health",
		"GET /ready",
		"HEAD /api/v1/stats",
		"HEAD /api/v1/users",
		"HEAD /api/v1/users/:id",
		"HEAD /api/v1/users/:id/tasks",
		"HEAD /health",
		"HEAD /ready",
		"POST /api/v1/tasks",
		"POST /api/v1/users",
		"PUT /api/v1/tasks/:id",
	}, got)
}

// GIVEN an unknown path or an unsupported method / THEN 404 ROUTE_NOT_FOUND or 405 METHOD_NOT_ALLOWED in the error envelope
func TestRouter_UnknownRouteAndMethod(t *testing.T) {
	s := newTestServer(t)

	requireError(t, s.doOutsideOperations(http.MethodGet, "/api/v1/unknown"), http.StatusNotFound, "ROUTE_NOT_FOUND", "Route not found")

	notAllowed := s.doOutsideOperations(http.MethodPatch, "/api/v1/users")
	requireError(t, notAllowed, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	assert.Equal(t, "OPTIONS, GET, HEAD, POST", notAllowed.header.Get("Allow"))

	requireError(t, s.doOutsideOperations(http.MethodPost, "/health"), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// GIVEN HEAD on an unknown path, on a real server / THEN 404 without a body
func TestRouter_HeadOnUnknownRouteHasNoBody(t *testing.T) {
	base := liveServer(t, newTestServer(t))

	r := call(t, http.MethodHead, base+"/api/v1/unknown", "", nil)

	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Empty(t, r.body)
}

// GIVEN a browser preflight from any origin / THEN CORS allows the five methods of the API
func TestRouter_CORSPreflightAllowsAnyOrigin(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks/1", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:5173")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPut)
	rec := httptest.NewRecorder()

	s.e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "*", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	assert.Equal(t, "GET,HEAD,POST,PUT,DELETE", rec.Header().Get(echo.HeaderAccessControlAllowMethods))
}

// GIVEN a simple cross-origin request / THEN the response carries the CORS origin header
func TestRouter_CORSSimpleRequest(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(echo.HeaderOrigin, "https://example.org")
	rec := httptest.NewRecorder()

	s.e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "*", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
}

// GIVEN a handler that panics behind the production middlewares / THEN 500 INTERNAL_ERROR without detail, and the server keeps serving
func TestRouter_RecoversFromPanics(t *testing.T) {
	s := newTestServer(t)
	s.e.GET("/panic", func(*echo.Context) error { panic("boom secret detail") })

	r := s.doRaw(http.MethodGet, "/panic", "")

	contract.ValidateErrorEnvelope(t, r.body)
	requireError(t, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	assert.NotContains(t, string(r.body), "boom")
	assert.Equal(t, http.StatusOK, s.do(http.MethodGet, "/health", "").code, "the server must survive the panic")
}

type healthJSON struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	ID      string `json:"id"`
}

// GIVEN a running API / WHEN GET /health (with or without query) / THEN the contract body with a fresh UUID each time
func TestHealth_ReturnsContractBody(t *testing.T) {
	s := newTestServer(t)

	var ids []string
	for _, path := range []string{"/health", "/health?foo=bar"} {
		r := s.do(http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, r.code, path)
		env := decode[healthJSON](t, r)
		assert.True(t, env.Success, path)
		assert.Equal(t, "Service is healthy", env.Message, path)
		assert.Equal(t, "ok", env.Data.Status, path)
		assert.Equal(t, "1.0.0", env.Data.Version, path)
		_, err := uuid.Parse(env.Data.ID)
		assert.NoError(t, err, "id must be a UUID (%s)", path)
		ids = append(ids, env.Data.ID)
	}
	assert.NotEqual(t, ids[0], ids[1], "each response carries a new id")
}

// GIVEN a JSON error body / THEN it is valid JSON (guards requireError against HTML error pages)
func TestRouter_ErrorBodiesAreJSON(t *testing.T) {
	s := newTestServer(t)

	r := s.do(http.MethodGet, "/api/v1/users/abc", "")

	assert.True(t, json.Valid(r.body), "body: %s", r.body)
	assert.Equal(t, echo.MIMEApplicationJSON, r.header.Get(echo.HeaderContentType))
}

// GIVEN a handler that returns another 4xx HTTP error / THEN its status is kept, in the envelope with INVALID_BODY
func TestErrorHandler_OtherClientErrorKeepsItsStatus(t *testing.T) {
	s := newTestServer(t)
	s.e.GET("/teapot", func(*echo.Context) error { return echo.NewHTTPError(http.StatusTeapot, "secret detail") })

	r := s.doRaw(http.MethodGet, "/teapot", "")

	contract.ValidateErrorEnvelope(t, r.body)
	requireError(t, r, http.StatusTeapot, "INVALID_BODY", "Invalid request")
	assert.NotContains(t, string(r.body), "secret detail")
}

// GIVEN a handler that already answered and then returns an error / THEN the answer is kept as is, nothing is appended
func TestErrorHandler_LeavesACommittedResponseAlone(t *testing.T) {
	s := newTestServer(t)
	s.e.GET("/committed", func(c *echo.Context) error {
		if err := c.String(http.StatusAccepted, "done"); err != nil {
			return err
		}
		return echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	})

	r := s.doRaw(http.MethodGet, "/committed", "")

	assert.Equal(t, http.StatusAccepted, r.code)
	assert.Equal(t, "done", string(r.body))
}
