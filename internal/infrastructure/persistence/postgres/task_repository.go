package postgres

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/models"
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
func (r *TaskRepository) Create(ctx context.Context, t *task.Task) (uint, error) {
	// Mapper l'entité domaine vers le modèle GORM
	gormTask := &models.Task{
		Title:       t.Title,
		Description: t.Description,
		UserID:      t.UserID,
		Completed:   t.Completed,
	}

	// Créer la tâche
	result := r.db.WithContext(ctx).Create(gormTask)
	if result.Error != nil {
		return 0, result.Error
	}

	return gormTask.ID, nil
}

// GetByID récupère une tâche par son ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uint) (*task.Task, error) {
	var gormTask models.Task
	result := r.db.WithContext(ctx).First(&gormTask, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, task.ErrTaskNotFound
		}
		return nil, result.Error
	}

	// Mapper le modèle GORM vers l'entité domaine
	return &task.Task{
		ID:          gormTask.ID,
		Title:       gormTask.Title,
		Description: gormTask.Description,
		UserID:      gormTask.UserID,
		Completed:   gormTask.Completed,
	}, nil
}

// ListByUserID récupère toutes les tâches d'un utilisateur.
func (r *TaskRepository) ListByUserID(ctx context.Context, userID uint) ([]*task.Task, error) {
	var gormTasks []models.Task
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&gormTasks)

	if result.Error != nil {
		return nil, result.Error
	}

	// Mapper les modèles GORM vers les entités domaine
	tasks := make([]*task.Task, len(gormTasks))
	for i, gt := range gormTasks {
		tasks[i] = &task.Task{
			ID:          gt.ID,
			Title:       gt.Title,
			Description: gt.Description,
			UserID:      gt.UserID,
			Completed:   gt.Completed,
		}
	}

	return tasks, nil
}

// Update met à jour une tâche existante.
func (r *TaskRepository) Update(ctx context.Context, t *task.Task) error {
	gormTask := &models.Task{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		UserID:      t.UserID,
		Completed:   t.Completed,
	}

	result := r.db.WithContext(ctx).Model(&models.Task{}).Where("id = ?", t.ID).Updates(gormTask)
	return result.Error
}

// Delete supprime une tâche.
func (r *TaskRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Task{}, id)
	return result.Error
}

// Count retourne le nombre total de tâches.
func (r *TaskRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Task{}).Count(&count)
	return count, result.Error
}
