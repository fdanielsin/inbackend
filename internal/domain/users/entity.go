// Package users contient l'entité, les erreurs et le port de persistance des utilisateurs.
package users

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// User est l'entité métier User indépendante de toute technologie.
// Elle représente l'invariant métier d'un utilisateur.
type User struct {
	ID    uint
	Name  string
	Email string
}

// NormalizeName retire les espaces de début et de fin du nom ; un nom vide ou
// fait d'espaces est refusé. Création et modification partagent cette règle.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrUserNameEmpty
	}
	return name, nil
}

// NewUser crée une nouvelle entité User avec validation basique.
// Le nom et l'email sont débarrassés des espaces de début et de fin, l'email
// est en plus passé en minuscules : c'est sa forme normalisée qui est unique.
func NewUser(id uint, name, email string) (*User, error) {
	name, err := NormalizeName(name)
	if err != nil {
		return nil, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, ErrUserEmailEmpty
	}
	if !hasValidEmailFormat(email) {
		return nil, ErrUserEmailInvalid
	}

	return &User{
		ID:    id,
		Name:  name,
		Email: email,
	}, nil
}

// Limites de longueur d'un email, en caractères Unicode (RFC 5321 et RFC 1035).
const (
	maxEmailLocalPartLength = 64
	maxDomainLabelLength    = 63
	minTLDLength            = 2
	maxTLDLength            = 24
	punycodePrefix          = "xn--"
)

// hasValidEmailFormat vérifie un seul « @ », une partie locale valide, un
// domaine d'au moins deux labels valides et un TLD valide, sans espace ni
// caractère de contrôle ; ce n'est pas une validation RFC 5322.
func hasValidEmailFormat(email string) bool {
	if strings.IndexFunc(email, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return false
	}
	local, domain, found := strings.Cut(email, "@")
	if !found {
		return false
	}
	if !isValidLocalPart(local) {
		return false
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if !isValidDomainLabel(label) {
			return false
		}
	}
	return isValidTLD(labels[len(labels)-1])
}

// isValidLocalPart : 1 à 64 caractères, sans point au début ni à la fin et sans
// deux points consécutifs (dot-atom de la RFC 5322).
func isValidLocalPart(local string) bool {
	if length := utf8.RuneCountInString(local); length < 1 || length > maxEmailLocalPartLength {
		return false
	}
	return !strings.HasPrefix(local, ".") && !strings.HasSuffix(local, ".") && !strings.Contains(local, "..")
}

// isValidDomainLabel : 1 à 63 caractères, lettres Unicode, signes combinants,
// chiffres ou tiret, sans tiret au début ni à la fin.
func isValidDomainLabel(label string) bool {
	if length := utf8.RuneCountInString(label); length < 1 || length > maxDomainLabelLength {
		return false
	}
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}
	return !strings.ContainsFunc(label, func(r rune) bool {
		return r != '-' && !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r)
	})
}

// isValidTLD accepte la forme punycode xn-- (24 caractères au plus ; le label
// est déjà validé, donc non vide après le préfixe) ou 2 à 24 lettres Unicode,
// première lettre non combinante, sans chiffre ni tiret. Le plus long TLD
// délégué fait 24 caractères.
func isValidTLD(tld string) bool {
	length := utf8.RuneCountInString(tld)
	if strings.HasPrefix(tld, punycodePrefix) {
		return length <= maxTLDLength
	}
	if length < minTLDLength || length > maxTLDLength {
		return false
	}
	first, _ := utf8.DecodeRuneInString(tld)
	return unicode.IsLetter(first) && !strings.ContainsFunc(tld, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsMark(r)
	})
}
