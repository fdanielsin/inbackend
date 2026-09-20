package postgres

import (
	"context"
	"log/slog"

	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/logger"
	"github.com/fdanielsin/inbackend/internal/models"
	"gorm.io/gorm"
)

// UserRepository implémente l'interface domain.user.Repository avec PostgreSQL/GORM.
type UserRepository struct {
	db  *gorm.DB
	log *slog.Logger
}

// NewUserRepository crée une nouvelle instance du repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db:  db,
		log: logger.GetSlog(),
	}
}

// Create crée un nouvel utilisateur et retourne l'ID généré.
func (r *UserRepository) Create(ctx context.Context, u *user.User) (uint, error) {
	r.log.Debug("Creating user", slog.String("email", u.Email), slog.String("name", u.Name))

	// Mapper l'entité domaine vers le modèle GORM
	gormUser := &models.User{
		Name:  u.Name,
		Email: u.Email,
	}

	// Créer l'utilisateur
	result := r.db.WithContext(ctx).Create(gormUser)
	if result.Error != nil {
		r.log.Error("Failed to create user", slog.String("error", result.Error.Error()))
		return 0, result.Error
	}

	r.log.Info("User created", slog.Int64("id", int64(gormUser.ID)), slog.String("email", u.Email))
	return gormUser.ID, nil
}

// GetByID récupère un utilisateur par son ID.
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*user.User, error) {
	r.log.Debug("Fetching user by ID", slog.Int64("id", int64(id)))

	var gormUser models.User
	result := r.db.WithContext(ctx).First(&gormUser, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			r.log.Info("User not found", slog.Int64("id", int64(id)))
			return nil, user.ErrUserNotFound
		}
		r.log.Error("Failed to fetch user", slog.Int64("id", int64(id)), slog.String("error", result.Error.Error()))
		return nil, result.Error
	}

	// Mapper le modèle GORM vers l'entité domaine
	return &user.User{
		ID:    gormUser.ID,
		Name:  gormUser.Name,
		Email: gormUser.Email,
	}, nil
}

// GetByEmail récupère un utilisateur par son email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var gormUser models.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&gormUser)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, user.ErrUserNotFound
		}
		return nil, result.Error
	}

	return &user.User{
		ID:    gormUser.ID,
		Name:  gormUser.Name,
		Email: gormUser.Email,
	}, nil
}

// List récupère tous les utilisateurs.
func (r *UserRepository) List(ctx context.Context) ([]*user.User, error) {
	var gormUsers []models.User
	result := r.db.WithContext(ctx).Find(&gormUsers)

	if result.Error != nil {
		return nil, result.Error
	}

	// Mapper les modèles GORM vers les entités domaine
	users := make([]*user.User, len(gormUsers))
	for i, gu := range gormUsers {
		users[i] = &user.User{
			ID:    gu.ID,
			Name:  gu.Name,
			Email: gu.Email,
		}
	}

	return users, nil
}

// Delete supprime un utilisateur.
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	return result.Error
}
