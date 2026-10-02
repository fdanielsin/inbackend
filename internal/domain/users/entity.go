// Package users contient l'entité, les erreurs et le port de persistance des utilisateurs.
package users

import (
	"strings"
	"unicode"
)

// User est l'entité métier User indépendante de toute technologie.
// Elle représente l'invariant métier d'un utilisateur.
type User struct {
	ID    uint
	Name  string
	Email string
}

// NewUser crée une nouvelle entité User avec validation basique.
// Le nom et l'email sont débarrassés des espaces de début et de fin, l'email
// est en plus passé en minuscules : c'est sa forme normalisée qui est unique.
func NewUser(id uint, name, email string) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrUserNameEmpty
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, ErrUserEmailEmpty
	}
	if !hasMinimalEmailFormat(email) {
		return nil, ErrUserEmailInvalid
	}

	return &User{
		ID:    id,
		Name:  name,
		Email: email,
	}, nil
}

// hasMinimalEmailFormat vérifie un seul « @ », une partie locale non vide, un
// domaine d'au moins deux labels non vides séparés par des points, et l'absence
// d'espace ou de caractère de contrôle ; ce n'est pas une validation RFC 5322.
func hasMinimalEmailFormat(email string) bool {
	if strings.IndexFunc(email, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return false
	}
	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" || strings.Contains(domain, "@") {
		return false
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" {
			return false
		}
	}
	return true
}
