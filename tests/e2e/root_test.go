package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type summaryJSON struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Links   map[string]string `json:"links"`
}

// keysOf returns the sorted keys of the JSON object found at the given path of the body.
func keysOf(t *testing.T, body []byte, path ...string) []string {
	t.Helper()
	var node any
	require.NoError(t, json.Unmarshal(body, &node))
	for _, step := range path {
		node = node.(map[string]any)[step]
	}
	var keys []string
	for key := range node.(map[string]any) {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sendRoot sends GET / with the given headers (validated against the contract).
func (s *testServer) sendRoot(headers map[string]string) response {
	s.t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	r := response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
	contract.ValidateResponse(s.t, req, r.code, r.header, r.body)
	return r
}

// GIVEN the docs option / WHEN GET / (with a query too) / THEN the summary: the contract envelope, exactly the fields of AVAIL-R10
func TestRoot_ServesTheSummary(t *testing.T) {
	s := newDocsServer(t)

	t.Run("AVAIL-R08: answers 200 with the name, the version and the four links (happy path)", func(t *testing.T) {
		r := s.in(t).do(http.MethodGet, "/", "")

		require.Equal(t, http.StatusOK, r.code)
		assert.Equal(t, echo.MIMEApplicationJSON, r.header.Get(echo.HeaderContentType))
		env := decode[summaryJSON](t, r)
		assert.True(t, env.Success)
		assert.Equal(t, "Welcome to the InDrive Backend API", env.Message)
		assert.Equal(t, "InDrive Backend API", env.Data.Name)
		assert.Equal(t, map[string]string{"docs": "/docs", "openapi": "/openapi.yaml", "health": "/health", "ready": "/ready"}, env.Data.Links)
	})

	t.Run("AVAIL-R09: the envelope has no other top-level field (limit)", func(t *testing.T) {
		r := s.in(t).do(http.MethodGet, "/", "")

		assert.Equal(t, []string{"data", "message", "success"}, keysOf(t, r.body))
	})

	t.Run("AVAIL-R10: data holds nothing else than the name, the version and the links (limit)", func(t *testing.T) {
		r := s.in(t).do(http.MethodGet, "/", "")

		assert.Equal(t, []string{"links", "name", "version"}, keysOf(t, r.body, "data"))
		assert.Equal(t, []string{"docs", "health", "openapi", "ready"}, keysOf(t, r.body, "data", "links"))
	})

	t.Run("AVAIL-R08: a query string is ignored (limit)", func(t *testing.T) {
		plain := s.in(t).do(http.MethodGet, "/", "")
		withQuery := s.in(t).do(http.MethodGet, "/?x=1", "")

		assert.Equal(t, http.StatusOK, withQuery.code)
		assert.JSONEq(t, string(plain.body), string(withQuery.body))
	})
}

// GIVEN the summary / WHEN each link is followed on the same origin / THEN it answers 200
func TestRoot_LinksPointToExistingRoutes(t *testing.T) {
	s := newDocsServer(t)
	env := decode[summaryJSON](t, s.do(http.MethodGet, "/", ""))
	require.Len(t, env.Data.Links, 4)

	for name, link := range env.Data.Links {
		t.Run("AVAIL-R12: link "+name+" answers 200 (happy path)", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, s.in(t).doRaw(http.MethodGet, link, "").code)
		})
	}
}

// GIVEN no docs option (API_DOCS_ENABLED=false or production) / WHEN GET / / THEN only /health and /ready, and /docs is still 404
func TestRoot_WithoutDocsOmitsTheDocumentationLinks(t *testing.T) {
	s := newTestServer(t)

	r := s.do(http.MethodGet, "/", "")

	require.Equal(t, http.StatusOK, r.code)
	env := decode[summaryJSON](t, r)
	assert.Equal(t, map[string]string{"health": "/health", "ready": "/ready"}, env.Data.Links, "AVAIL-R12: no link to a 404 (boundary)")
	assert.Equal(t, []string{"health", "ready"}, keysOf(t, r.body, "data", "links"), "absent, not null or empty")
	requireError(t, s.doOutsideOperations(http.MethodGet, "/docs"), http.StatusNotFound, "ROUTE_NOT_FOUND", "Route not found")
}

// GIVEN Accept-Language fr-FR, en, de and none / WHEN GET / / THEN data is byte for byte the same, only the message follows the language
func TestRoot_OnlyTheMessageFollowsTheLanguage(t *testing.T) {
	s := newDocsServer(t)
	cases := []struct {
		header, message, language string
	}{
		{"fr-FR", "Bienvenue sur l'API InDrive Backend", "fr"},
		{"en", "Welcome to the InDrive Backend API", "en"},
		{"de", "Welcome to the InDrive Backend API", "en"},
		{"", "Welcome to the InDrive Backend API", "en"},
	}

	var first json.RawMessage
	for _, c := range cases {
		t.Run("AVAIL-R13: Accept-Language '"+c.header+"' (happy path)", func(t *testing.T) {
			headers := map[string]string{}
			if c.header != "" {
				headers[headerAcceptLanguage] = c.header
			}
			r := s.in(t).sendRoot(headers)

			var env struct {
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(r.body, &env))
			assert.Equal(t, c.message, env.Message)
			requireLanguageHeaders(t, r, c.language)
			if first == nil {
				first = env.Data
			}
			assert.Equal(t, string(first), string(env.Data), "data must not depend on the language")
		})
	}
}

// GIVEN the database is stopped / WHEN GET / / THEN the same summary, while /ready answers 503
func TestRoot_AnswersWithoutTheDatabase(t *testing.T) {
	s := newDocsServer(t)
	before := s.do(http.MethodGet, "/", "")
	s.closeDatabase()

	after := s.do(http.MethodGet, "/", "")

	assert.Equal(t, http.StatusOK, after.code, "AVAIL-R10: the summary does not read the database (exception)")
	assert.Equal(t, string(before.body), string(after.body))
	assert.Equal(t, http.StatusServiceUnavailable, s.do(http.MethodGet, "/ready", "").code)
}

// GIVEN hostile Host and X-Forwarded-* headers / WHEN GET / / THEN no link and no field carries them, and no link has a scheme or a host
func TestRoot_NeverBuildsLinksFromTheRequest(t *testing.T) {
	s := newDocsServer(t)

	r := s.sendRoot(map[string]string{
		"X-Forwarded-Host":  "evil.example",
		"X-Forwarded-Proto": "https",
		"X-Forwarded-For":   "203.0.113.9",
	})
	hostile := httptest.NewRequest(http.MethodGet, "http://evil.example/", nil)
	hostile.Host = "evil.example"
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, hostile)

	for _, body := range []string{string(r.body), rec.Body.String()} {
		assert.NotContains(t, body, "evil.example", "AVAIL-R12: the Host is never echoed (sad path)")
		assert.NotContains(t, body, "203.0.113.9")
		assert.NotContains(t, body, "://")
	}
	for _, link := range decode[summaryJSON](t, r).Data.Links {
		assert.True(t, strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "//"), link)
	}
	assert.Equal(t, http.StatusOK, rec.Code)
}

// GIVEN HEAD / / THEN the status and the headers of GET /, no body ; GIVEN POST, PUT, DELETE and PATCH / THEN 405 with Allow
func TestRoot_OtherMethods(t *testing.T) {
	s := newDocsServer(t)

	t.Run("AVAIL-R14: HEAD answers like GET without a body (happy path)", func(t *testing.T) {
		url := liveServer(t, s.in(t))
		get := callOperation(t, http.MethodGet, url+"/", "", nil)
		head := callOperation(t, http.MethodHead, url+"/", "", nil)

		assert.Equal(t, get.status, head.status)
		assert.Empty(t, head.body)
		assert.Equal(t, get.header.Get(echo.HeaderContentType), head.header.Get(echo.HeaderContentType))
		assert.Equal(t, "no-store", head.header.Get(echo.HeaderCacheControl))
		assert.Equal(t, "en", head.header.Get("Content-Language"))
		assert.Contains(t, head.header.Values("Vary"), "Accept-Language")
	})

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run("AVAIL-R14: "+method+" answers 405 with Allow (sad path)", func(t *testing.T) {
			r := s.in(t).doOutsideOperations(method, "/")

			requireError(t, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
			assert.Contains(t, r.header.Get(echo.HeaderAllow), http.MethodGet)
			assert.Contains(t, r.header.Get(echo.HeaderAllow), http.MethodHead)
		})
	}
}

// GIVEN GET / / THEN Cache-Control: no-store
func TestRoot_IsNeverCached(t *testing.T) {
	s := newDocsServer(t)

	r := s.do(http.MethodGet, "/", "")

	assert.Equal(t, "no-store", r.header.Get(echo.HeaderCacheControl), "AVAIL-R15 (happy path)")
}

// GIVEN the neighbouring paths of the root / THEN they stay 404 ROUTE_NOT_FOUND
func TestRoot_NeighbouringPathsAreUnknown(t *testing.T) {
	s := newDocsServer(t)

	for _, path := range []string{"/api", "/api/v1", "/index.html", "/favicon.ico"} {
		t.Run("AVAIL-R08: "+path+" is unknown (sad path)", func(t *testing.T) {
			requireError(t, s.in(t).doOutsideOperations(http.MethodGet, path), http.StatusNotFound, "ROUTE_NOT_FOUND", "Route not found")
		})
	}
}

// GIVEN a release that changes the version / THEN the summary, /health, the constant and info.version of the contract agree
func TestRoot_VersionIsTheOneOfHealthAndTheContract(t *testing.T) {
	s := newDocsServer(t)

	summary := decode[summaryJSON](t, s.do(http.MethodGet, "/", "")).Data.Version
	health := decode[healthJSON](t, s.do(http.MethodGet, "/health", "")).Data.Version

	assert.Equal(t, shared.APIVersion, summary, "AVAIL-R11: one value in the code (happy path)")
	assert.Equal(t, summary, health)
	assert.Equal(t, contract.Version(), summary, "the code follows info.version of the contract")
}

// GIVEN the real binary / WHEN ENV and API_DOCS_ENABLED vary / THEN the summary links /docs and /openapi.yaml exactly when they exist
func TestRoot_RealBinaryLinksFollowTheDocsSetting(t *testing.T) {
	cases := []struct {
		env, value string
		links      map[string]string
	}{
		{"development", "", map[string]string{"docs": "/docs", "openapi": "/openapi.yaml", "health": "/health", "ready": "/ready"}},
		{"production", "", map[string]string{"health": "/health", "ready": "/ready"}},
		{"development", "false", map[string]string{"health": "/health", "ready": "/ready"}},
		{"production", "true", map[string]string{"docs": "/docs", "openapi": "/openapi.yaml", "health": "/health", "ready": "/ready"}},
	}
	for _, c := range cases {
		t.Run("AVAIL-R12: ENV="+c.env+" API_DOCS_ENABLED="+c.value+" (happy path)", func(t *testing.T) {
			server := startServer(t, "ENV="+c.env, "API_DOCS_ENABLED="+c.value, "CORS_ALLOWED_ORIGINS=https://app.example")

			got := callOperation(t, http.MethodGet, "http://"+server.addr+"/", "", nil)

			require.Equal(t, http.StatusOK, got.status)
			var env struct {
				Data summaryJSON `json:"data"`
			}
			require.NoError(t, json.Unmarshal(got.body, &env))
			assert.Equal(t, c.links, env.Data.Links)
			for _, link := range c.links {
				assert.Equal(t, http.StatusOK, getStatus(t, "http://"+server.addr+link), link)
			}
		})
	}
}
