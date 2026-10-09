package domain

import (
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repeat(char string, count int) string { return strings.Repeat(char, count) }

// GIVEN: Emails at the exact accepted limits and with accepted shapes / WHEN: NewUser / THEN: accepted
func TestNewUser_ValidEmailFormats_AreAccepted(t *testing.T) {
	for name, email := range map[string]string{
		"local part of 64":             repeat("a", 64) + "@x.io",
		"label of 63":                  "a@" + repeat("b", 63) + ".io",
		"TLD of 24 letters":            "a@b." + repeat("c", 24),
		"TLD of 2 letters":             "a@b.io",
		"xn-- TLD":                     "a@b.xn--p1ai",
		"xn-- TLD of 24":               "a@b.xn--" + repeat("c", 20),
		"cyrillic IDN":                 "ivan@пример.рф",
		"IDN TLD with combining marks": "a@b.भारत",
		"digits and inner hyphens":     "a@b-1.c-2.io",
		"digit-only label":             "a@123.io",
		"four labels":                  "a@a.b.c.io",
		"specials in the local part":   "a.b+c_d-e@x.io",
		"63 characters, not 63 bytes":  "a@" + repeat("é", 63) + ".fr",
		"64 characters, not 64 bytes":  repeat("é", 64) + "@x.io",
		"upper case is lower-cased":    "ADA@EXAMPLE.COM",
		"punycode label in the middle": "a@xn--e1afmkfd.io",
	} {
		t.Run(name, func(t *testing.T) {
			u, err := users.NewUser(0, "Ada", email)

			require.NoError(t, err, email)
			assert.Equal(t, strings.ToLower(email), u.Email)
		})
	}
}

// GIVEN: Emails just past a limit or breaking a label rule / WHEN: NewUser / THEN: ErrUserEmailInvalid
func TestNewUser_InvalidEmailFormats_AreRejected(t *testing.T) {
	for name, email := range map[string]string{
		"local part of 65":           repeat("a", 65) + "@x.io",
		"65 characters, not bytes":   repeat("é", 65) + "@x.io",
		"label of 64":                "a@" + repeat("b", 64) + ".io",
		"label of 64 in the middle":  "a@x." + repeat("b", 64) + ".io",
		"TLD of 25 letters":          "a@b." + repeat("c", 25),
		"TLD of 1 letter":            "a@b.c",
		"TLD with a digit":           "a@b.c0m",
		"digit-only TLD":             "a@b.12",
		"TLD with a hyphen":          "a@b.co-op",
		"TLD starting with a mark":   "a@b.́a",
		"xn-- alone":                 "a@b.xn--",
		"xn-- TLD of 25":             "a@b.xn--" + repeat("c", 21),
		"underscore in a label":      "a@b_c.com",
		"label starting with hyphen": "a@-a.com",
		"label ending with hyphen":   "a@a-.com",
		"TLD starting with hyphen":   "a@b.-com",
		"TLD ending with hyphen":     "a@b.com-",
		"only a hyphen label":        "a@-.com",
		"symbol in a label":          "a@b!c.com",
		"emoji in a label":           "a@b🚀.com",
		"label of 64 non-ASCII":      "a@" + repeat("é", 64) + ".fr",
		"one label only":             "a@com",
		"empty label":                "a@b..com",
		"trailing dot":               "a@b.com.",
		"space inside":               "a b@x.io",
	} {
		t.Run(name, func(t *testing.T) {
			u, err := users.NewUser(0, "Ada", email)

			assert.Nil(t, u, email)
			assert.ErrorIs(t, err, users.ErrUserEmailInvalid, "%q", email)
		})
	}
}
