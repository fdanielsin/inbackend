package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contractFile is the contract as committed, which the binary must serve unchanged.
func contractFile(t *testing.T) []byte {
	t.Helper()
	spec, err := os.ReadFile(filepath.Join("..", "..", "docs", "specs", "api", "openapi.yaml"))
	require.NoError(t, err)
	return spec
}

func newDocsServer(t *testing.T) *testServer {
	t.Helper()
	return newTestServer(t, app.WithAPIDocs(contractFile(t)))
}

// cspDirectives splits a Content-Security-Policy into its directives.
func cspDirectives(t *testing.T, policy string) map[string][]string {
	t.Helper()
	directives := map[string][]string{}
	for _, part := range strings.Split(policy, ";") {
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		require.NotContains(t, directives, fields[0], "directive %s given twice", fields[0])
		directives[fields[0]] = fields[1:]
	}
	return directives
}

// GIVEN the docs option / WHEN GET /openapi.yaml / THEN the committed contract, byte for byte, as YAML that kin-openapi loads and validates
func TestAPIDocs_ServesTheContract(t *testing.T) {
	s := newDocsServer(t)

	for _, path := range []string{"/openapi.yaml", "/openapi.yaml?download=1"} {
		t.Run(path, func(t *testing.T) {
			r := s.in(t).do(http.MethodGet, path, "")

			require.Equal(t, http.StatusOK, r.code, path)
			assert.Equal(t, "application/yaml", r.header.Get(echo.HeaderContentType), path)
			assert.Equal(t, "nosniff", r.header.Get("X-Content-Type-Options"), path)
			assert.NotEmpty(t, r.header.Get(echo.HeaderXRequestID), path)
			assert.Equal(t, contractFile(t), r.body, "%s serves the contract unchanged", path)
		})
	}
	doc, err := openapi3.NewLoader().LoadFromData(s.do(http.MethodGet, "/openapi.yaml", "").body)
	require.NoError(t, err)
	require.NoError(t, doc.Validate(context.Background()))
}

var (
	externalAsset = regexp.MustCompile(`<(?:script|link)\b[^>]*(?:src|href)="(https://[^"]+)"[^>]*>`)
	inlineScript  = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

// GIVEN the docs option / WHEN GET /docs / THEN Swagger UI at an exact version, each CDN file with its
// integrity, and a CSP that allows only that CDN and the hash of the one inline script
func TestAPIDocs_SwaggerPageIsLockedDown(t *testing.T) {
	s := newDocsServer(t)

	r := s.do(http.MethodGet, "/docs", "")

	require.Equal(t, http.StatusOK, r.code)
	assert.True(t, strings.HasPrefix(r.header.Get(echo.HeaderContentType), "text/html"), r.header.Get(echo.HeaderContentType))
	assert.Equal(t, "nosniff", r.header.Get("X-Content-Type-Options"))
	page := string(r.body)
	assert.Contains(t, page, `url: "/openapi.yaml"`)

	assets := externalAsset.FindAllStringSubmatch(page, -1)
	require.Len(t, assets, 2, "the Swagger UI script and style sheet")
	for _, asset := range assets {
		tag, url := asset[0], asset[1]
		assert.Regexp(t, `^https://cdn\.jsdelivr\.net/npm/swagger-ui-dist@\d+\.\d+\.\d+/`, url, "an exact version")
		assert.Regexp(t, `integrity="sha384-[A-Za-z0-9+/]{64}"`, tag, "subresource integrity")
		assert.Contains(t, tag, `crossorigin="anonymous"`)
	}

	inline := inlineScript.FindAllStringSubmatch(page, -1)
	require.Len(t, inline, 1, "one inline script")
	sum := sha256.Sum256([]byte(inline[0][1]))
	scriptHash := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"

	csp := cspDirectives(t, r.header.Get("Content-Security-Policy"))
	assert.Equal(t, []string{"'none'"}, csp["default-src"])
	assert.ElementsMatch(t, []string{"https://cdn.jsdelivr.net", scriptHash}, csp["script-src"],
		"only the CDN and the hash of the inline script as served")
	assert.Equal(t, []string{"'none'"}, csp["frame-ancestors"])
	for directive, sources := range csp {
		assert.NotContains(t, sources, "'unsafe-eval'", directive)
		if directive != "style-src" {
			assert.NotContains(t, sources, "'unsafe-inline'", "%s: inline is allowed for styles only", directive)
		}
	}
}

// GIVEN the docs option / WHEN "Try it out" calls the first server of the served contract
// THEN that server is the origin of the page, whatever its host and port, the only one its CSP lets it connect to
func TestAPIDocs_TryItOutTargetsTheDocsOrigin(t *testing.T) {
	s := newDocsServer(t)

	doc, err := openapi3.NewLoader().LoadFromData(s.do(http.MethodGet, "/openapi.yaml", "").body)
	require.NoError(t, err)
	require.NotEmpty(t, doc.Servers, "Swagger UI needs a server to call")
	assert.Equal(t, "/", doc.Servers[0].URL, "the default server is relative to the page")

	csp := cspDirectives(t, s.do(http.MethodGet, "/docs", "").header.Get("Content-Security-Policy"))
	assert.Equal(t, []string{"'self'"}, csp["connect-src"], "the page calls its own origin only")
}

// GIVEN the docs option / WHEN another method than GET / THEN 405 METHOD_NOT_ALLOWED in the envelope, Allow listing GET
func TestAPIDocs_OtherMethodsAreNotAllowed(t *testing.T) {
	s := newDocsServer(t)

	for _, path := range []string{"/openapi.yaml", "/docs"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			t.Run(method+" "+path, func(t *testing.T) {
				r := s.in(t).doOutsideOperations(method, path)

				requireError(t, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
				assert.Contains(t, r.header.Get(echo.HeaderAllow), http.MethodGet, "%s %s", method, path)
			})
		}
	}
}

// GIVEN the docs option / WHEN a neighbouring path / THEN 404 ROUTE_NOT_FOUND
func TestAPIDocs_NeighbouringPathsAreUnknown(t *testing.T) {
	s := newDocsServer(t)

	for _, path := range []string{"/docs/", "/openapi.yml", "/openapi.json", "/docs/index.html"} {
		t.Run(path, func(t *testing.T) {
			requireError(t, s.in(t).doOutsideOperations(http.MethodGet, path), http.StatusNotFound, "ROUTE_NOT_FOUND", "Route not found")
		})
	}
}

// GIVEN the docs option / WHEN asked in English, in French or without preference
// THEN Content-Language is the real language of the content: /openapi.yaml in French, /docs in English
func TestAPIDocs_ContentLanguageIsTheRealOne(t *testing.T) {
	s := newDocsServer(t)
	contentLanguage := map[string]string{"/openapi.yaml": "fr", "/docs": "en"}

	for path, want := range contentLanguage {
		for _, accept := range []string{"", "en", "fr", "fr-FR,en;q=0.5"} {
			t.Run(fmt.Sprintf("%s Accept-Language %q", path, accept), func(t *testing.T) {
				var r response
				if accept == "" {
					r = s.in(t).sendWithLanguages(http.MethodGet, path, "", true)
				} else {
					r = s.in(t).sendWithLanguages(http.MethodGet, path, "", true, accept)
				}

				require.Equal(t, http.StatusOK, r.code)
				assert.Equal(t, want, r.header.Get("Content-Language"), "%s with Accept-Language %q", path, accept)
				assert.Contains(t, varyValues(r), "accept-language", "%s: Vary stays as on every response", path)
			})
		}
	}
}

// GIVEN a router without the docs option / THEN /openapi.yaml and /docs do not exist (404), and with it exactly these two paths more, for GET and HEAD
func TestAPIDocs_RoutesExistOnlyWithTheOption(t *testing.T) {
	routes := func(s *testServer) []string {
		var got []string
		for _, r := range s.e.Router().Routes() {
			got = append(got, r.Method+" "+r.Path)
		}
		sort.Strings(got)
		return got
	}
	without := newTestServer(t)
	with := newDocsServer(t)

	for _, path := range []string{"/openapi.yaml", "/docs"} {
		t.Run("without the option "+path, func(t *testing.T) {
			r := without.in(t).doRaw(http.MethodGet, path, "")
			contract.ValidateErrorEnvelope(t, r.body)
			requireError(t, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route not found")
		})
	}
	assert.Len(t, routes(without), 16)
	assert.ElementsMatch(t, append(routes(without), "GET /docs", "GET /openapi.yaml", "HEAD /docs", "HEAD /openapi.yaml"), routes(with))
}

// docsStatus starts the real binary with the given settings and returns the status of GET /docs and the
// body of GET /openapi.yaml.
func docsStatus(t *testing.T, settings ...string) (int, []byte) {
	t.Helper()
	server := startServer(t, settings...)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+server.addr+"/openapi.yaml", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	spec, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return getStatus(t, "http://"+server.addr+"/docs"), spec
}

// GIVEN the real binary / WHEN ENV and API_DOCS_ENABLED vary / THEN the docs exist unless ENV=production,
// an explicit value (strconv.ParseBool) always wins, and the embedded contract is the committed one
func TestAPIDocs_RealBinaryFollowsEnvAndSetting(t *testing.T) {
	cases := []struct {
		env, value string
		enabled    bool
	}{
		{"", "", true},
		{"development", "", true},
		{"production", "", false},
		{"development", "false", false},
		{"production", "true", true},
		{"production", "1", true},
		{"", "0", false},
		{"production", "TRUE", true},
		{"development", "f", false},
	}
	for _, c := range cases {
		t.Run("ENV="+c.env+" API_DOCS_ENABLED="+c.value, func(t *testing.T) {
			status, spec := docsStatus(t, "ENV="+c.env, "API_DOCS_ENABLED="+c.value)

			if c.enabled {
				assert.Equal(t, http.StatusOK, status)
				assert.Equal(t, contractFile(t), spec, "the binary embeds the committed contract")
			} else {
				assert.Equal(t, http.StatusNotFound, status)
			}
		})
	}
}

// GIVEN an API_DOCS_ENABLED value that strconv.ParseBool refuses / WHEN the server starts
// THEN it exits with code 1, says why, and neither listens nor migrates the database
func TestAPIDocs_InvalidSettingStopsTheStart(t *testing.T) {
	pg.Require(t)
	bin := buildServer(t)
	for _, value := range []string{"yes", "2", "maybe", " true", "on"} {
		t.Run(value, func(t *testing.T) {
			dsn := pg.NewEmptyDatabase(t)

			code, out := runUntilExit(t, bin, serverEnv("DATABASE_URL="+dsn, "PORT="+freePort(t), "API_DOCS_ENABLED="+value))

			assert.Equal(t, 1, code)
			assert.Contains(t, out, "API_DOCS_ENABLED must be true or false")
			assert.NotContains(t, out, "Starting server")
			assert.False(t, testsupport.OpenDSN(t, dsn).Migrator().HasTable("schema_migrations"), "no migration before the setting is checked")
		})
	}
}
