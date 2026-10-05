package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const corsProductionError = "CORS_ALLOWED_ORIGINS must list the allowed origins when ENV=production (not empty, not *)"

// isolatedServerEnv is serverEnv without the inherited ENV and CORS_ALLOWED_ORIGINS, so that a case
// can leave them really absent.
func isolatedServerEnv(settings ...string) []string {
	var env []string
	for _, kv := range serverEnv() {
		if !strings.HasPrefix(kv, "ENV=") && !strings.HasPrefix(kv, "CORS_ALLOWED_ORIGINS=") {
			env = append(env, kv)
		}
	}
	return append(env, settings...)
}

// allowedOrigin returns the Access-Control-Allow-Origin answered to a request from origin.
func allowedOrigin(t *testing.T, addr, origin string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/health", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", origin)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.Header.Get("Access-Control-Allow-Origin")
}

// GIVEN ENV=production and CORS_ALLOWED_ORIGINS absent, empty or allowing any origin ("*")
// WHEN the server starts / THEN it exits with code 1, says why without echoing the value,
// and neither listens nor migrates the database
func TestStartup_ProductionRequiresExplicitCORSOrigins(t *testing.T) {
	pg.Require(t)
	bin := buildServer(t)
	for _, tc := range []struct {
		name    string
		setting []string
	}{
		{"variable absent", nil},
		{"empty", []string{"CORS_ALLOWED_ORIGINS="}},
		{"any origin", []string{"CORS_ALLOWED_ORIGINS=*"}},
		{"an origin and any origin", []string{"CORS_ALLOWED_ORIGINS=https://a.example,*"}},
		{"separators only", []string{"CORS_ALLOWED_ORIGINS=,"}},
		{"any origin among spaces", []string{"CORS_ALLOWED_ORIGINS= https://a.example , * "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dsn := pg.NewEmptyDatabase(t)

			code, out := runUntilExit(t, bin, isolatedServerEnv(append([]string{"ENV=production", "DATABASE_URL=" + dsn, "PORT=" + freePort(t)}, tc.setting...)...))

			assert.Equal(t, 1, code)
			assert.Contains(t, out, corsProductionError)
			assert.NotContains(t, out, "https://a.example", "the value received is not echoed")
			assert.NotContains(t, out, "Starting server")
			assert.False(t, testsupport.OpenDSN(t, dsn).Migrator().HasTable("schema_migrations"), "no migration before the setting is checked")
		})
	}
}

// GIVEN ENV=production and an explicit list of origins / WHEN the server starts
// THEN it serves, and only the listed origins are allowed
func TestStartup_ProductionServesOnlyListedOrigins(t *testing.T) {
	for _, tc := range []struct {
		value  string
		listed []string
	}{
		{"https://app.example", []string{"https://app.example"}},
		{"https://app.example, https://admin.example", []string{"https://app.example", "https://admin.example"}},
		{"http://localhost:5173,http://[::1]:3000", []string{"http://localhost:5173", "http://[::1]:3000"}},
	} {
		t.Run(tc.value, func(t *testing.T) {
			s := startServer(t, "ENV=production", "CORS_ALLOWED_ORIGINS="+tc.value)

			for _, origin := range tc.listed {
				assert.Equal(t, origin, allowedOrigin(t, s.addr, origin))
			}
			assert.Empty(t, allowedOrigin(t, s.addr, "https://evil.example"))
		})
	}
}

// GIVEN an ENV other than exactly "production" (case included) and CORS_ALLOWED_ORIGINS "*" or empty
// WHEN the server starts / THEN it serves and allows any origin, as before
func TestStartup_OutsideProductionKeepsAnyOrigin(t *testing.T) {
	for _, env := range []string{"Production", "development", ""} {
		for _, value := range []string{"*", ""} {
			t.Run("ENV="+env+" CORS_ALLOWED_ORIGINS="+value, func(t *testing.T) {
				s := startServer(t, "ENV="+env, "CORS_ALLOWED_ORIGINS="+value)

				assert.Equal(t, "*", allowedOrigin(t, s.addr, "https://any.example"))
			})
		}
	}
}

// GIVEN ENV=production without CORS origins and a DATABASE_URL that is not a URL
// WHEN the server starts / THEN the CORS setting is refused first, before any database setting is read
func TestStartup_CORSCheckedBeforeTheDatabase(t *testing.T) {
	bin := buildServer(t)

	code, out := runUntilExit(t, bin, isolatedServerEnv("ENV=production", "DATABASE_URL=host=localhost password=secretpw dbname=x", "PORT="+freePort(t)))

	assert.Equal(t, 1, code)
	assert.Contains(t, out, corsProductionError)
	assert.NotContains(t, out, "DATABASE_URL must be a postgres:// URL")
	assert.NotContains(t, out, "secretpw")
}

// GIVEN a malformed origin, in production or not, with or without DATABASE_URL / WHEN the server starts
// THEN it exits with code 1 before reading DATABASE_URL, names the rank of the entry and the reason,
// and does not echo the value
func TestStartup_RefusesMalformedCORSOrigins(t *testing.T) {
	bin := buildServer(t)
	for _, tc := range []struct {
		name, env, value, want string
	}{
		{"trailing slash in production", "production", "https://a.example,https://b.example/", "entry 2 is not a valid origin: must not contain a path, trailing slash, query or fragment"},
		{"trailing slash outside production", "development", "https://b.example/", "entry 1 is not a valid origin: must not contain a path, trailing slash, query or fragment"},
		{"scheme missing", "production", "app.example", "entry 1 is not a valid origin: must start with lowercase http:// or https://"},
		{"host in capitals", "production", " https://App.example ", "entry 1 is not a valid origin: host is not a valid lowercase ASCII host name or bracketed IP address"},
		{"subdomain pattern", "production", "https://*.example", "entry 1 is not a valid origin: wildcard host patterns are not supported: list each origin"},
		{"default port written", "production", "https://a.example:443", "entry 1 is not a valid origin: default port must be omitted (80 for http, 443 for https)"},
		{"port out of range", "development", "https://a.example:65536", "entry 1 is not a valid origin: port must be a number from 1 to 65535 without leading zero"},
		{"userinfo", "production", "https://user:secretpw@a.example", "entry 1 is not a valid origin: must not contain credentials (userinfo)"},
		{"separators do not shift the rank of the next entry", "production", ",, https://a.example, a.example", "entry 2 is not a valid origin: must start with lowercase http:// or https://"},
		{"any origin among others outside production", "development", "https://a.example,*", "entry 2 is not a valid origin: the wildcard * cannot be combined with other origins"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, out := runUntilExit(t, bin, isolatedServerEnv("ENV="+tc.env, "CORS_ALLOWED_ORIGINS="+tc.value, "PORT="+freePort(t)))

			assert.Equal(t, 1, code)
			assert.Contains(t, out, "CORS_ALLOWED_ORIGINS "+tc.want)
			assert.NotContains(t, out, "DATABASE_URL not set", "the origins are checked before DATABASE_URL")
			assert.NotContains(t, out, "secretpw", "the value received is not echoed")
			assert.NotContains(t, out, "Starting server")
		})
	}
}

// GIVEN a malformed origin and a database / WHEN the server starts / THEN the database is not migrated
func TestStartup_MalformedCORSOriginLeavesTheDatabaseUntouched(t *testing.T) {
	pg.Require(t)
	bin := buildServer(t)
	dsn := pg.NewEmptyDatabase(t)

	code, out := runUntilExit(t, bin, isolatedServerEnv("ENV=production", "CORS_ALLOWED_ORIGINS=https://a.example/", "DATABASE_URL="+dsn, "PORT="+freePort(t)))

	assert.Equal(t, 1, code)
	assert.Contains(t, out, "CORS_ALLOWED_ORIGINS entry 1 is not a valid origin")
	assert.False(t, testsupport.OpenDSN(t, dsn).Migrator().HasTable("schema_migrations"))
}
