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
