package pgstore

import (
	"context"
	"errors"

	"github.com/fdanielsin/inbackend/internal/domain/users"
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
func (r *UserRepository) Create(ctx context.Context, u *users.User) (uint, error) {
	gormUser := &UserModel{
		Name:  u.Name,
		Email: u.Email,
	}

	result := r.db.WithContext(ctx).Create(gormUser)
	if result.Error != nil {
		if isConstraintViolation(result.Error, pgUniqueViolation, uniqueEmailIndex) {
			return 0, users.ErrUserEmailTaken
		}
		return 0, result.Error
	}

	return gormUser.ID, nil
}

// GetByID récupère un utilisateur par son ID.
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*users.User, error) {
	var gormUser UserModel
	result := r.db.WithContext(ctx).First(&gormUser, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, users.ErrUserNotFound
		}
		return nil, result.Error
	}

	return &users.User{
		ID:    gormUser.ID,
		Name:  gormUser.Name,
		Email: gormUser.Email,
	}, nil
}

// GetByEmail récupère un utilisateur par son email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	var gormUser UserModel
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&gormUser)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, users.ErrUserNotFound
		}
		return nil, result.Error
	}

	return &users.User{
		ID:    gormUser.ID,
		Name:  gormUser.Name,
		Email: gormUser.Email,
	}, nil
}

// List récupère tous les utilisateurs.
func (r *UserRepository) List(ctx context.Context) ([]*users.User, error) {
	var gormUsers []UserModel
	result := r.db.WithContext(ctx).Order("id ASC").Find(&gormUsers)

	if result.Error != nil {
		return nil, result.Error
	}

	listed := make([]*users.User, len(gormUsers))
	for i, gu := range gormUsers {
		listed[i] = &users.User{
			ID:    gu.ID,
			Name:  gu.Name,
			Email: gu.Email,
		}
	}

	return listed, nil
}

// Count retourne le nombre total d'utilisateurs.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count)
	return count, result.Error
}

// Update remplace le nom d'un utilisateur.
//
// Updates avec Select("name") n'écrit que le nom et ne crée jamais de ligne
// (Save ferait un upsert qui recréerait un utilisateur supprimé) ; l'e-mail et
// l'identifiant ne sont jamais écrits. RowsAffected == 0 : l'utilisateur
// n'existe pas (ou plus, en cas de suppression concurrente).
func (r *UserRepository) Update(ctx context.Context, u *users.User) error {
	result := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ?", u.ID).
		Select("name").
		Updates(UserModel{Name: u.Name})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return users.ErrUserNotFound
	}
	return nil
}

// Delete supprime un utilisateur ; ses tâches suivent par la clé étrangère
// (ON DELETE CASCADE), dans la même instruction : tout ou rien, sans
// transaction applicative. RowsAffected == 0 : il n'existe pas (ou plus).
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&UserModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return users.ErrUserNotFound
	}
	return nil
}
