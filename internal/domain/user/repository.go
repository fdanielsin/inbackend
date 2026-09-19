package user

import "context"

// Repository définit le contrat pour l'accès aux données User.
// L'implémentation concrète (PostgreSQL/GORM) sera en infrastructure.
type Repository interface {
	// Create crée un nouvel utilisateur et retourne l'ID généré.
	Create(ctx context.Context, user *User) (uint, error)

	// GetByID récupère un utilisateur par son ID.
	GetByID(ctx context.Context, id uint) (*User, error)

	// GetByEmail récupère un utilisateur par son email.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// List récupère tous les utilisateurs.
	List(ctx context.Context) ([]*User, error)

	// Delete supprime un utilisateur.
	Delete(ctx context.Context, id uint) error
}
