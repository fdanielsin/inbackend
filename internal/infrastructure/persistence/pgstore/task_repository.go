package pgstore

import (
	"context"
	"errors"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"gorm.io/gorm"
)

// TaskRepository implémente l'interface domain.task.Repository avec PostgreSQL/GORM.
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository crée une nouvelle instance du repository.
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create crée une nouvelle tâche et retourne l'ID généré.
func (r *TaskRepository) Create(ctx context.Context, t *tasks.Task) (uint, error) {
	gormTask := &TaskModel{
		Title:       t.Title,
		Description: t.Description,
		UserID:      t.UserID,
		Completed:   t.Completed,
	}

	result := r.db.WithContext(ctx).Create(gormTask)
	if result.Error != nil {
		if isConstraintViolation(result.Error, pgForeignKeyViolation, taskUserForeignKey) {
			return 0, users.ErrUserNotFound
		}
		return 0, result.Error
	}

	return gormTask.ID, nil
}

// GetByID récupère une tâche par son ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uint) (*tasks.Task, error) {
	var gormTask TaskModel
	result := r.db.WithContext(ctx).First(&gormTask, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, tasks.ErrTaskNotFound
		}
		return nil, result.Error
	}

	return &tasks.Task{
		ID:          gormTask.ID,
		Title:       gormTask.Title,
		Description: gormTask.Description,
		UserID:      gormTask.UserID,
		Completed:   gormTask.Completed,
	}, nil
}

// ListByUserID récupère toutes les tâches d'un utilisateur.
func (r *TaskRepository) ListByUserID(ctx context.Context, userID uint) ([]*tasks.Task, error) {
	var gormTasks []TaskModel
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&gormTasks)

	if result.Error != nil {
		return nil, result.Error
	}

	listed := make([]*tasks.Task, len(gormTasks))
	for i, gt := range gormTasks {
		listed[i] = &tasks.Task{
			ID:          gt.ID,
			Title:       gt.Title,
			Description: gt.Description,
			UserID:      gt.UserID,
			Completed:   gt.Completed,
		}
	}

	return listed, nil
}

// Update met à jour une tâche existante.
//
// Select() force l'écriture de title/description/completed même à leur
// valeur zéro (chaîne vide, false) : sans lui, GORM ignorerait silencieusement
// ces champs dans la clause UPDATE lorsqu'on lui passe un struct.
// UserID n'est volontairement pas dans la liste Select : ce chemin ne doit
// jamais réattribuer une tâche à un autre utilisateur.
//
// RowsAffected == 0 signifie que l'ID ne correspond à aucune ligne (tâche
// déjà supprimée, y compris via une suppression concurrente entre GetByID et
// cet appel) ; on le traite comme "non trouvée" plutôt que comme un succès
// silencieux.
func (r *TaskRepository) Update(ctx context.Context, t *tasks.Task) error {
	result := r.db.WithContext(ctx).
		Model(&TaskModel{}).
		Where("id = ?", t.ID).
		Select("title", "description", "completed").
		Updates(TaskModel{
			Title:       t.Title,
			Description: t.Description,
			Completed:   t.Completed,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return tasks.ErrTaskNotFound
	}
	return nil
}

// Delete supprime une tâche ; RowsAffected == 0 signifie qu'elle n'existe pas
// (ou plus, en cas de suppression concurrente).
func (r *TaskRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&TaskModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return tasks.ErrTaskNotFound
	}
	return nil
}

// Count retourne le nombre total de tâches.
func (r *TaskRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&TaskModel{}).Count(&count)
	return count, result.Error
}
