package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const headerAcceptLanguage = "Accept-Language"

// sendWithLanguages sends a request carrying the given Accept-Language header values
// (none when empty); validate checks the response against openapi.yaml.
func (s *testServer) sendWithLanguages(method, path, body string, validate bool, languages ...string) response {
	s.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	for _, language := range languages {
		req.Header.Add(headerAcceptLanguage, language)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	r := response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
	if validate {
		contract.ValidateResponse(s.t, req, r.code, r.header, r.body)
	}
	return r
}

// requireLanguageHeaders checks Content-Language and that Vary lists Accept-Language.
func requireLanguageHeaders(t *testing.T, r response, language string) {
	t.Helper()
	assert.Equal(t, language, r.header.Get("Content-Language"))
	assert.Contains(t, varyValues(r), "accept-language", "Vary: %v", r.header.Values("Vary"))
}

// varyValues returns the Vary tokens in lower case, whether sent in one header or several.
func varyValues(r response) []string {
	var values []string
	for _, header := range r.header.Values("Vary") {
		for _, token := range strings.Split(header, ",") {
			values = append(values, strings.ToLower(strings.TrimSpace(token)))
		}
	}
	return values
}

// GIVEN Accept-Language values (RFC 9110 §12.5.4, RFC 4647 basic filtering) / WHEN an error is answered
// THEN the language is the acceptable range with the highest weight that is served (fr or en), en otherwise,
// never a 400 because of the header
func TestLanguage_NegotiationFollowsAcceptLanguage(t *testing.T) {
	s := newTestServer(t)
	english := decode[any](t, s.sendWithLanguages(http.MethodGet, "/api/v1/users/abc", "", true)).Error
	require.Equal(t, "Invalid user ID", english, "English stays the default")

	cases := []struct {
		header string
		want   string
	}{
		{"fr", "fr"},
		{"fr-FR", "fr"},
		{"fr-CA", "fr"},
		{"FR", "fr"},
		{"Fr-ch", "fr"},
		{"en", "en"},
		{"en-US", "en"},
		{"de", "en"},
		{"de;q=1, fr;q=0.5", "fr"},
		{"fr;q=0.5, en;q=0.9", "en"},
		{"en;q=0.5, fr;q=0.9", "fr"},
		{"fr, en", "fr"},
		{"en, fr", "en"},
		{"fr;q=0.8, en;q=0.8", "fr"},
		{"en;q=0.8, fr;q=0.8", "en"},
		{"fr;q=0", "en"},
		{"fr;q=0.001", "fr"},
		{"fr;q=1", "fr"},
		{"fr;q=1.000", "fr"},
		{"*", "en"},
		{"!!!", "en"},
		{"fr;q=abc", "en"},
		{"fr;q=1.5", "en"},
		{"fr_FR", "en"},
		{",,,", "en"},
		{" ", "en"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%q", c.header), func(t *testing.T) {
			r := s.sendWithLanguages(http.MethodGet, "/api/v1/users/abc", "", true, c.header)

			require.Equal(t, http.StatusBadRequest, r.code, "the header never changes the status")
			env := decode[any](t, r)
			assert.Equal(t, "INVALID_ID", env.ErrorCode)
			requireLanguageHeaders(t, r, c.want)
			if c.want == "en" {
				assert.Equal(t, english, env.Error)
			} else {
				assert.NotEqual(t, english, env.Error, "a French message")
				assert.NotEmpty(t, env.Error)
			}
		})
	}
}

// GIVEN two Accept-Language header lines / THEN they are one list (RFC 9110 §5.3): de then fr gives fr
func TestLanguage_SeveralHeaderLinesFormOneList(t *testing.T) {
	s := newTestServer(t)

	r := s.sendWithLanguages(http.MethodGet, "/api/v1/users/abc", "", true, "de", "fr;q=0.5")

	requireLanguageHeaders(t, r, "fr")
}

// GIVEN an 8 KiB header of 200 unserved ranges, or of garbage / THEN English, quickly, without 400
func TestLanguage_VeryLongHeaderFallsBackToEnglish(t *testing.T) {
	s := newTestServer(t)
	ranges := make([]string, 200)
	for i := range ranges {
		ranges[i] = fmt.Sprintf("x%d-abcdefghijklmnopqrstuvwxyz;q=0.%03d", i, i+1)
	}
	for name, header := range map[string]string{
		"many ranges": strings.Join(ranges, ", "),
		"garbage":     strings.Repeat("é;,=", 2048),
	} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			r := s.sendWithLanguages(http.MethodGet, "/api/v1/users/abc", "", true, header)

			assert.Less(t, time.Since(start), 200*time.Millisecond)
			require.Equal(t, http.StatusBadRequest, r.code)
			requireLanguageHeaders(t, r, "en")
		})
	}
}

// GIVEN every kind of response (success, handler errors, router errors, 413, 500, 503, probes)
// WHEN asked in English (no header) and in French / THEN same status and error_code, the message in the
// language, Content-Language and Vary: Accept-Language, and a response valid against the contract
func TestLanguage_EveryResponseIsLocalized(t *testing.T) {
	big := `{"name":"` + strings.Repeat("a", 1<<20) + `","email":"a@b.co"}`
	requests := []struct {
		name, method, path, body string
		inContract               bool
		closeDB                  bool
	}{
		{"list users", http.MethodGet, "/api/v1/users", "", true, false},
		{"create user", http.MethodPost, "/api/v1/users", `{"name":"Ada","email":"ada@example.com"}`, true, false},
		{"validation", http.MethodPost, "/api/v1/users", `{"name":"","email":"ada@example.com"}`, true, false},
		{"invalid body", http.MethodPost, "/api/v1/users", `{`, true, false},
		{"unknown user", http.MethodGet, "/api/v1/users/999", "", true, false},
		{"unknown task", http.MethodDelete, "/api/v1/tasks/999", "", true, false},
		{"create task", http.MethodPost, "/api/v1/tasks", `{"user_id":1,"title":"Buy milk"}`, true, false},
		{"stats", http.MethodGet, "/api/v1/stats", "", true, false},
		{"health", http.MethodGet, "/health", "", true, false},
		{"ready", http.MethodGet, "/ready", "", true, false},
		{"unknown route", http.MethodGet, "/api/v1/unknown", "", false, false},
		{"method not allowed", http.MethodPost, "/health", "", false, false},
		{"body too large", http.MethodPost, "/api/v1/users", big, true, false},
		{"internal error", http.MethodGet, "/api/v1/users", "", true, true},
		{"not ready", http.MethodGet, "/ready", "", true, true},
	}
	for _, rq := range requests {
		t.Run(rq.name, func(t *testing.T) {
			answers := map[string]response{}
			for _, language := range []string{"en", "fr"} {
				s := newTestServer(t)
				if rq.name == "create task" {
					s.createUser("Owner", "owner@example.com")
				}
				if rq.closeDB {
					s.closeDatabase()
				}
				var r response
				if language == "en" {
					r = s.sendWithLanguages(rq.method, rq.path, rq.body, rq.inContract)
				} else {
					r = s.sendWithLanguages(rq.method, rq.path, rq.body, rq.inContract, "fr")
				}
				if !rq.inContract {
					contract.ValidateErrorEnvelope(t, r.body)
				}
				requireLanguageHeaders(t, r, language)
				answers[language] = r
			}

			en, fr := answers["en"], answers["fr"]
			require.Equal(t, en.code, fr.code, "the language never changes the status")
			enEnv, frEnv := decode[any](t, en), decode[any](t, fr)
			assert.Equal(t, enEnv.ErrorCode, frEnv.ErrorCode, "error_code does not depend on the language")
			assert.Equal(t, enEnv.Success, frEnv.Success)
			if enEnv.Error != "" {
				assert.NotEqual(t, enEnv.Error, frEnv.Error, "error translated")
				assert.NotEmpty(t, frEnv.Error)
			}
			if enEnv.Message != "" {
				assert.NotEqual(t, enEnv.Message, frEnv.Message, "message translated")
				assert.NotEmpty(t, frEnv.Message)
			}
		})
	}
}

// GIVEN a router error, a HEAD on an unknown route and a CORS preflight / WHEN asked in French
// THEN Content-Language and Vary are set too, and Vary keeps the values set by CORS
func TestLanguage_HeadersOutsideTheOperations(t *testing.T) {
	s := newTestServer(t)

	head := s.sendWithLanguages(http.MethodHead, "/api/v1/unknown", "", false, "fr")
	assert.Equal(t, http.StatusNotFound, head.code)
	assert.Empty(t, head.body)
	requireLanguageHeaders(t, head, "fr")

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks/1", nil)
	req.Header.Set(echo.HeaderOrigin, "https://app.example.com")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPut)
	req.Header.Set(headerAcceptLanguage, "fr")
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	preflight := response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
	assert.Equal(t, http.StatusNoContent, preflight.code)
	requireLanguageHeaders(t, preflight, "fr")
	assert.Contains(t, varyValues(preflight), "origin", "CORS values are kept: %v", preflight.header.Values("Vary"))
}

// GIVEN the database is down / WHEN asked in French / THEN a generic 500, without any technical detail
func TestLanguage_InternalErrorStaysGenericInFrench(t *testing.T) {
	s := newTestServer(t)
	s.closeDatabase()

	r := s.sendWithLanguages(http.MethodGet, "/api/v1/users", "", true, "fr")

	require.Equal(t, http.StatusInternalServerError, r.code)
	env := decode[any](t, r)
	assert.Equal(t, "INTERNAL_ERROR", env.ErrorCode)
	for _, detail := range []string{"database", "closed", "sql", "testpass", "testuser", "postgres"} {
		assert.NotContains(t, strings.ToLower(string(r.body)), detail)
	}
}

// GIVEN the same requests in English and in French / THEN the logs are the same: log messages are not translated
func TestLanguage_LogsDoNotDependOnTheLanguage(t *testing.T) {
	linesFor := func(language string) []string {
		logs := captureLogs(t)
		s := newTestServer(t)
		for _, path := range []string{"/api/v1/users/abc", "/api/v1/users/999", "/api/v1/unknown"} {
			if language == "" {
				s.sendWithLanguages(http.MethodGet, path, "", false)
			} else {
				s.sendWithLanguages(http.MethodGet, path, "", false, language)
			}
		}
		var lines []string
		for _, msg := range []string{"Invalid user ID", "Request", "Request failed", "Fetching user"} {
			for _, line := range logLines(t, logs, msg) {
				delete(line, "time")
				delete(line, "request_id")
				delete(line, "latency")
				lines = append(lines, fmt.Sprint(line))
			}
		}
		return lines
	}

	english := linesFor("")
	require.NotEmpty(t, english)
	assert.Equal(t, english, linesFor("fr"))
}

// GIVEN the real binary / WHEN an error is requested in French / THEN a French body with its headers
func TestLanguage_RealBinaryAnswersInFrench(t *testing.T) {
	server := startServer(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+server.addr+"/api/v1/users/abc", nil)
	require.NoError(t, err)
	req.Header.Set(headerAcceptLanguage, "fr-FR,fr;q=0.9,en;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "fr", resp.Header.Get("Content-Language"))
	assert.Contains(t, strings.ToLower(strings.Join(resp.Header.Values("Vary"), ",")), "accept-language")
}

// GIVEN a body over 1 MiB sent without Content-Length (chunked), refused while it is read rather than by
// Echo's limit / WHEN asked in French / THEN the 413 is in French too
func TestLanguage_ChunkedBodyTooLargeIsLocalized(t *testing.T) {
	s := newTestServer(t)
	body := `{"name":"` + strings.Repeat("a", 1<<20) + `","email":"a@b.co"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(headerAcceptLanguage, "fr")
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	rec := httptest.NewRecorder()

	s.e.ServeHTTP(rec, req)

	r := response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
	contract.ValidateResponse(t, req, r.code, r.header, r.body)
	requireError(t, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", shared.Message(shared.LangFrench, shared.MsgPayloadTooLarge))
	requireLanguageHeaders(t, r, "fr")
}
