package domain

import (
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN names with and without surrounding spaces / WHEN NormalizeName / THEN the spaces are removed and the inside kept
func TestNormalizeName_TrimsSurroundingSpaces(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"USER-R17: a name is kept as is (happy path)", "Grace", "Grace"},
		{"USER-R17: spaces around a name are removed (happy path)", "  Grace Hopper \t", "Grace Hopper"},
		{"USER-R17: a one character name is accepted (boundary)", "G", "G"},
		{"USER-R17: spaces inside a name are kept (limit)", "Grace  Hopper", "Grace  Hopper"},
		{"USER-R17: 100 emojis are accepted (limit)", strings.Repeat("😀", 100), strings.Repeat("😀", 100)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := users.NormalizeName(tc.in)

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// GIVEN an empty name or one made of spaces / WHEN NormalizeName / THEN ErrUserNameEmpty
func TestNormalizeName_RefusesAnEmptyName(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"USER-R17: an empty name is refused (sad path)", ""},
		{"USER-R17: a name made of spaces is refused (sad path)", " \t\n "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := users.NormalizeName(tc.in)

			assert.Empty(t, got)
			assert.ErrorIs(t, err, users.ErrUserNameEmpty)
		})
	}
}
