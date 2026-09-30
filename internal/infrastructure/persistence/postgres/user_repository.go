package postgres

import (
	"context"
	"errors"

	"github.com/fdanielsin/inbackend/internal/domain/user"
	"gorm.io/gorm"
)

// UserRepository implémente l'interface domain.user.Repository avec PostgreSQL/GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository crée une nouvelle instance du repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create crée un nouvel utilisateur et retourne l'ID généré.
func (r *UserRepository) Create(ctx context.Context, u *user.User) (uint, error) {
	gormUser := &UserModel{
		Name:  u.Name,
		Email: u.Email,
	}

	result := r.db.WithContext(ctx).Create(gormUser)
	if result.Error != nil {
		if isConstraintViolation(result.Error, pgUniqueViolation, uniqueEmailIndex) {
			return 0, user.ErrUserEmailTaken
		}
		return 0, result.Error
	}

	return gormUser.ID, nil
}

// GetByID récupère un utilisateur par son ID.
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*user.User, error) {
	var gormUser UserModel
	result := r.db.WithContext(ctx).First(&gormUser, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
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

// GetByEmail récupère un utilisateur par son email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var gormUser UserModel
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&gormUser)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
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
	var gormUsers []UserModel
	result := r.db.WithContext(ctx).Find(&gormUsers)

	if result.Error != nil {
		return nil, result.Error
	}

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

// Count retourne le nombre total d'utilisateurs.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count)
	return count, result.Error
}
