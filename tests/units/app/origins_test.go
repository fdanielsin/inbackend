package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	reasonScheme   = "must start with lowercase http:// or https://"
	reasonUserinfo = "must not contain credentials (userinfo)"
	reasonPath     = "must not contain a path, trailing slash, query or fragment"
	reasonSpace    = "must not contain spaces or control characters"
	reasonNoHost   = "host is missing"
	reasonPattern  = "wildcard host patterns are not supported: list each origin"
	reasonHost     = "host is not a valid lowercase ASCII host name or bracketed IP address"
	reasonPort     = "port must be a number from 1 to 65535 without leading zero"
	reasonDefault  = "default port must be omitted (80 for http, 443 for https)"
	reasonNumeric  = "a host whose last label is numeric must be a canonical IPv4 address (four decimal bytes from 0 to 255, no leading zero)"
	reasonMapped   = "IPv4-mapped IPv6 addresses are not supported"
	reasonMixed    = "the wildcard * cannot be combined with other origins"
)

func refusal(rank, reason string) string {
	return "CORS_ALLOWED_ORIGINS entry " + rank + " is not a valid origin: " + reason
}

func label(n int) string { return strings.Repeat("a", n) }

// GIVEN well-formed origins / WHEN validated / THEN they are accepted
func TestValidateAllowedOrigins_AcceptsWellFormedOrigins(t *testing.T) {
	longestHost := strings.Join([]string{label(63), label(63), label(63), label(61)}, ".") // 253 characters
	for _, origin := range []string{
		"https://app.example.com",
		"http://app.example.com",
		"http://localhost:5173",
		"https://localhost",
		"http://127.0.0.1:8080",
		"http://0.0.0.0",           // 0 alone is canonical
		"http://255.255.255.255",   // highest byte
		"https://10.20.30.40:8443", // IPv4 with a port
		"http://1.2.3.4a",          // last label not numeric: a name, not an IPv4
		"https://123abc",
		"https://example.0xg", // "0x" followed by a non-hexadecimal digit is not a number
		"http://[::1]:3000",
		"https://[2001:db8::1]",
		"https://a",
		"https://az09-za90.example", // every edge of the allowed alphabet
		"https://my-app.example.com:8443",
		"http://example.com:443",    // 443 is not the default port of http
		"https://example.com:80",    // 80 is not the default port of https
		"https://example.com:1",     // lowest port
		"https://example.com:65535", // highest port
		"https://" + label(63) + ".example.com",
		"https://" + longestHost,
		"*",
	} {
		t.Run(origin, func(t *testing.T) {
			require.NoError(t, app.ValidateAllowedOrigins([]string{origin}))
		})
	}
	require.NoError(t, app.ValidateAllowedOrigins([]string{"https://a.example", "http://localhost:5173"}), "several origins")
	require.NoError(t, app.ValidateAllowedOrigins(nil), "no origin: nothing to refuse")
}

// GIVEN a malformed origin / WHEN validated / THEN it is refused with its rank and the reason, never its value
func TestValidateAllowedOrigins_RefusesMalformedOrigins(t *testing.T) {
	cases := []struct {
		name, origin, reason string
	}{
		{"no scheme", "app.example.com", reasonScheme},
		{"other scheme", "ftp://app.example.com", reasonScheme},
		{"scheme in capitals", "HTTPS://app.example.com", reasonScheme},
		{"scheme only", "https://", reasonNoHost},
		{"scheme without slashes", "https:app.example.com", reasonScheme},
		{"one slash", "https:/app.example.com", reasonScheme},
		{"leading space inside", " https://x", reasonScheme},
		{"trailing slash", "https://app.example.com/", reasonPath},
		{"path", "https://app.example.com/app", reasonPath},
		{"query", "https://app.example.com?a=1", reasonPath},
		{"fragment", "https://app.example.com#top", reasonPath},
		{"backslash", `https://app.example.com\app`, reasonPath},
		{"userinfo", "https://user:secret@app.example.com", reasonUserinfo},
		{"userinfo and path", "https://secret@app.example.com/p", reasonUserinfo},
		{"inner space", "https://app .example.com", reasonSpace},
		{"tab", "https://app.example.com\t", reasonSpace},
		{"newline", "https://app.example.com\n", reasonSpace},
		{"delete character", "https://app.example.com\x7f", reasonSpace},
		{"null byte", "https://app\x00.example.com", reasonSpace},
		{"host in capitals", "https://App.Example.com", reasonHost},
		{"host with underscore", "https://my_app.example.com", reasonHost},
		{"non-ASCII host", "https://café.example.com", reasonHost},
		{"empty label", "https://app..example.com", reasonHost},
		{"trailing dot", "https://app.example.com.", reasonHost},
		{"leading dot", "https://.example.com", reasonHost},
		{"label starting with hyphen", "https://-app.example.com", reasonHost},
		{"label ending with hyphen", "https://app-.example.com", reasonHost},
		{"label of 64 characters", "https://" + label(64) + ".example.com", reasonHost},
		{"host of 254 characters", "https://" + strings.Join([]string{label(63), label(63), label(63), label(62)}, "."), reasonHost},
		{"IPv6 not in brackets", "http://::1", reasonNoHost},
		{"IPv6 in capitals", "http://[2001:DB8::1]", reasonHost},
		{"IPv6 not compressed", "http://[0:0:0:0:0:0:0:1]", reasonHost},
		{"IPv6 with zone", "http://[fe80::1%25eth0]", reasonHost},
		{"IPv4 in brackets", "http://[127.0.0.1]", reasonHost},
		{"IPv6 unclosed", "http://[::1", reasonHost},
		{"IPv6 followed by junk", "http://[::1]x", reasonHost},
		{"IPv4 with three bytes", "http://1.2.3", reasonNumeric},
		{"IPv4 with five bytes", "http://1.2.3.4.5", reasonNumeric},
		{"IPv4 with leading zero (octal for a browser)", "http://010.0.0.1", reasonNumeric},
		{"IPv4 with a lone leading zero pair", "http://1.2.3.00", reasonNumeric},
		{"IPv4 in hexadecimal", "http://0x7f.0.0.1", reasonNumeric},
		{"IPv4 byte above 255", "http://999.1.1.1", reasonNumeric},
		{"IPv4 byte 256", "http://1.2.3.256", reasonNumeric},
		{"name ending in a number", "https://example.123", reasonNumeric},
		{"host made of digits", "https://123", reasonNumeric},
		{"name ending in 0x", "https://example.0x", reasonNumeric},
		{"name ending in hexadecimal", "https://example.0x1f", reasonNumeric},
		{"IPv4-mapped IPv6 as Go writes it", "http://[::ffff:1.2.3.4]", reasonMapped},
		{"IPv4-mapped IPv6 as a browser writes it", "http://[::ffff:102:304]", reasonMapped},
		{"wildcard subdomain", "https://*.example.com", reasonPattern},
		{"wildcard host", "https://*", reasonPattern},
		{"wildcard in middle", "https://app.*.com", reasonPattern},
		{"wildcard port", "https://example.com:*", reasonPort},
		{"empty port", "https://example.com:", reasonPort},
		{"port zero", "https://example.com:0", reasonPort},
		{"port with leading zero", "https://example.com:08080", reasonPort},
		{"port 65536", "https://example.com:65536", reasonPort},
		{"huge port", "https://example.com:99999999999999999999", reasonPort},
		{"negative port", "https://example.com:-1", reasonPort},
		{"port with letters", "https://example.com:80a", reasonPort},
		{"two ports", "https://example.com:80:81", reasonPort},
		{"explicit default https port", "https://example.com:443", reasonDefault},
		{"explicit default http port", "http://example.com:80", reasonDefault},
		{"explicit default port on IPv6", "http://[::1]:80", reasonDefault},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := app.ValidateAllowedOrigins([]string{"https://ok.example", tc.origin})

			require.Error(t, err)
			require.EqualError(t, err, refusal("2", tc.reason))
			assert.NotContains(t, err.Error(), "secret", "the value received is not echoed")
			assert.NotContains(t, err.Error(), tc.origin, "the value received is not echoed")
		})
	}
}

// GIVEN empty items between commas / WHEN validated / THEN they are ignored but counted in the rank,
// as an operator counts the comma-separated positions of the variable
func TestValidateAllowedOrigins_RankCountsEveryCommaSeparatedPosition(t *testing.T) {
	entries := app.SplitAllowedOrigins(",, https://a.example, a.example")

	require.Equal(t, []string{"", "", "https://a.example", "a.example"}, entries)
	require.EqualError(t, app.ValidateAllowedOrigins(entries), refusal("4", reasonScheme))
	require.EqualError(t, app.ValidateAllowedOrigins(app.SplitAllowedOrigins(" , https://a.example/")), refusal("2", reasonPath))
	require.EqualError(t, app.ValidateAllowedOrigins(app.SplitAllowedOrigins(",https://a.example,*")), refusal("3", reasonMixed))
	require.NoError(t, app.ValidateAllowedOrigins(app.SplitAllowedOrigins(",,")), "nothing to refuse")
	require.NoError(t, app.ValidateAllowedOrigins(app.SplitAllowedOrigins(" , * ,")), "a sole wildcard among separators")
}

// GIVEN several faulty origins / WHEN validated / THEN the first one is reported, with its rank in the list
func TestValidateAllowedOrigins_ReportsTheFirstFaultWithItsRank(t *testing.T) {
	err := app.ValidateAllowedOrigins([]string{"https://a.example", "https://b.example", "https://c.example/", "http://d.example:80"})

	require.EqualError(t, err, refusal("3", reasonPath))
}

// GIVEN "*" / WHEN it stands alone or with other origins / THEN only the sole "*" is accepted
func TestValidateAllowedOrigins_WildcardMustBeAlone(t *testing.T) {
	require.NoError(t, app.ValidateAllowedOrigins([]string{"*"}))
	require.EqualError(t, app.ValidateAllowedOrigins([]string{"https://a.example", "*"}), refusal("2", reasonMixed))
	require.EqualError(t, app.ValidateAllowedOrigins([]string{"*", "https://a.example"}), refusal("1", reasonMixed))
	require.EqualError(t, app.ValidateAllowedOrigins([]string{"*", "*"}), refusal("1", reasonMixed))
}

// GIVEN origins accepted by the validation / WHEN given to Echo's CORS middleware / THEN it builds without panic
// and allows exactly the listed origin: nothing the validation accepts is rejected or reshaped by Echo
func TestValidateAllowedOrigins_AcceptedOriginsWorkWithEcho(t *testing.T) {
	for _, origin := range []string{"https://app.example.com", "http://localhost:5173", "http://[::1]:3000", "https://example.com:80"} {
		t.Run(origin, func(t *testing.T) {
			require.NoError(t, app.ValidateAllowedOrigins([]string{origin}))

			allowed, other := echoAnswer(t, []string{origin}, origin), echoAnswer(t, []string{origin}, "https://evil.example")

			assert.Equal(t, origin, allowed)
			assert.Empty(t, other)
		})
	}
}

// GIVEN Echo v5.4.0 / WHEN fed what the validation refuses / THEN Echo panics (trailing slash), or silently
// accepts a pattern that matches no subdomain, or matches without case: the validation catches them at startup
func TestEcho_BehaviourBehindTheRefusedOrigins(t *testing.T) {
	t.Run("a pattern is read as a literal origin: no subdomain matches", func(t *testing.T) {
		assert.Empty(t, echoAnswer(t, []string{"https://*.example.com"}, "https://a.example.com"))
	})
	t.Run("a trailing slash makes the middleware panic", func(t *testing.T) {
		assert.Panics(t, func() { middleware.CORS("https://app.example/") })
	})
	t.Run("the comparison ignores case and answers the configured value", func(t *testing.T) {
		assert.Equal(t, "https://App.example", echoAnswer(t, []string{"https://App.example"}, "https://app.example"))
	})
	t.Run("a wildcard among origins allows any origin", func(t *testing.T) {
		assert.Equal(t, "*", echoAnswer(t, []string{"*"}, "https://evil.example"))
		assert.Equal(t, "*", echoAnswer(t, []string{"https://a.example", "*"}, "https://evil.example"))
	})
}

// echoAnswer returns the Access-Control-Allow-Origin that Echo's CORS middleware, configured with
// configured, answers to a request from requestOrigin.
func echoAnswer(t *testing.T, configured []string, requestOrigin string) string {
	t.Helper()
	e := echo.New()
	e.Use(middleware.CORS(configured...))
	e.GET("/", func(c *echo.Context) error { return c.NoContent(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", requestOrigin)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	return rec.Header().Get("Access-Control-Allow-Origin")
}
