package app

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A browser sends its origin in the form the WHATWG URL parser serializes
// (https://url.spec.whatwg.org/#host-parsing): a host whose last label is a number is
// parsed as an IPv4 address and rewritten (1.2.3 -> 1.2.0.3, 010.0.0.1 -> 8.0.0.1,
// 0x7f.0.0.1 -> 127.0.0.1) or refused (999.1.1.1, example.123), and an IPv6 address is
// written with hexadecimal groups only (::ffff:1.2.3.4 -> ::ffff:102:304). An origin
// written any other way never matches a request, so it is refused at startup.

// GIVEN a host in the form a browser sends (canonical IPv4, name with numeric labels
// before the last one) / WHEN validated / THEN it is accepted
func TestValidateAllowedOrigins_AcceptsHostsInTheBrowserForm(t *testing.T) {
	for _, origin := range []string{
		"http://0.0.0.0",              // lowest octets
		"http://255.255.255.255:8080", // highest octets
		"http://192.168.1.10",
		"https://10.0.0.1",
		"https://123.example.com", // a numeric label that is not the last one
		"https://1a.example",
		"https://example.com1", // last label not a number
	} {
		t.Run(origin, func(t *testing.T) {
			require.NoError(t, app.ValidateAllowedOrigins([]string{origin}))
		})
	}
}

// GIVEN a host whose last label is a number but which is not a canonical IPv4 address, or an
// IPv6 address written with an embedded IPv4 / WHEN validated / THEN it is refused with its
// rank, without echoing the value
func TestValidateAllowedOrigins_RefusesHostsABrowserNeverSends(t *testing.T) {
	for _, tc := range []struct{ name, origin string }{
		{"IPv4 with three parts", "http://1.2.3"},
		{"IPv4 with one part", "http://1"},
		{"IPv4 as a single 32-bit number", "http://4294967295"},
		{"IPv4 with five parts", "http://1.2.3.4.5"},
		{"octal octet", "http://010.0.0.1"},
		{"octet with a leading zero, last", "http://1.2.3.04"},
		{"hexadecimal octet", "http://0x7f.0.0.1"},
		{"octet above 255", "http://999.1.1.1"},
		{"octet 256", "http://256.0.0.1"},
		{"name ending in a number", "https://example.123"},
		{"name ending in a hexadecimal number", "https://example.0x1f"},
		{"IPv4-mapped IPv6, dotted", "http://[::ffff:1.2.3.4]"},
		{"IPv4-mapped IPv6, hexadecimal", "http://[::ffff:102:304]"},
		{"IPv4-compatible IPv6, dotted", "http://[::1.2.3.4]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := app.ValidateAllowedOrigins([]string{tc.origin})

			require.Error(t, err)
			assert.Contains(t, err.Error(), refusal("1", ""), "the rank and the refusal are given")
			assert.NotContains(t, err.Error(), tc.origin, "the value received is not echoed")
		})
	}
}
