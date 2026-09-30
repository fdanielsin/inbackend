package task

import "context"

// Repository définit le contrat pour l'accès aux données Task.
// L'implémentation concrète (PostgreSQL/GORM) sera en infrastructure.
type Repository interface {
	// Create crée une nouvelle tâche et retourne l'ID généré.
	Create(ctx context.Context, task *Task) (uint, error)

	// GetByID récupère une tâche par son ID.
	GetByID(ctx context.Context, id uint) (*Task, error)

	// ListByUserID récupère toutes les tâches d'un utilisateur.
	ListByUserID(ctx context.Context, userID uint) ([]*Task, error)

	// Update met à jour une tâche existante.
	Update(ctx context.Context, task *Task) error

	// Delete supprime une tâche ; ErrTaskNotFound si elle n'existe pas.
	Delete(ctx context.Context, id uint) error

	// Count retourne le nombre total de tâches.
	Count(ctx context.Context) (int64, error)
}
