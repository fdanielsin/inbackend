// Package user contient l'entité, les erreurs et le port de persistance des utilisateurs.
package user

// User est l'entité métier User indépendante de toute technologie.
// Elle représente l'invariant métier d'un utilisateur.
type User struct {
	ID    uint
	Name  string
	Email string
}

// NewUser crée une nouvelle entité User avec validation basique.
func NewUser(id uint, name, email string) (*User, error) {
	if name == "" {
		return nil, ErrUserNameEmpty
	}
	if email == "" {
		return nil, ErrUserEmailEmpty
	}

	return &User{
		ID:    id,
		Name:  name,
		Email: email,
	}, nil
}
