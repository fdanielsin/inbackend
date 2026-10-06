package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// GIVEN CORS_ALLOWED_ORIGINS with empty or blank positions before a malformed origin / WHEN the
// server starts / THEN the rank in the message is the position between commas, as the operator
// counts it in the value, empty positions included
func TestStartup_CORSRankCountsEveryCommaSeparatedPosition(t *testing.T) {
	bin := buildServer(t)
	for _, tc := range []struct {
		name, env, value, want string
	}{
		{"two empty positions first", "production", ",, https://a.example, a.example", "CORS_ALLOWED_ORIGINS entry 4 is not a valid origin"},
		{"blank position in the middle", "production", "https://a.example, ,https://b.example/", "CORS_ALLOWED_ORIGINS entry 3 is not a valid origin"},
		{"one empty position first", "development", ",https://a.example/", "CORS_ALLOWED_ORIGINS entry 2 is not a valid origin"},
		{"wildcard after an empty position", "development", "https://a.example,,*", "CORS_ALLOWED_ORIGINS entry 3 is not a valid origin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, out := runUntilExit(t, bin, isolatedServerEnv("ENV="+tc.env, "CORS_ALLOWED_ORIGINS="+tc.value, "PORT="+freePort(t)))

			assert.Equal(t, 1, code)
			assert.Contains(t, out, tc.want)
			assert.NotContains(t, out, "Starting server")
		})
	}
}
