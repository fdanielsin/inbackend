package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
)

// ListUserTasksRequest est le DTO de la requête ListUserTasks.
type ListUserTasksRequest struct {
	UserID uint
}

// ListUserTasksResponse est la réponse de la requête ListUserTasks.
type ListUserTasksResponse struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	UserID      uint   `json:"user_id"`
	Completed   bool   `json:"completed"`
}

// ListUserTasksHandler exécute la logique de récupération des tâches d'un utilisateur.
type ListUserTasksHandler struct {
	repo tasks.Repository
}

// NewListUserTasksHandler crée une nouvelle instance du handler.
func NewListUserTasksHandler(repo tasks.Repository) *ListUserTasksHandler {
	return &ListUserTasksHandler{repo: repo}
}

// Execute exécute la requête ListUserTasks.
func (h *ListUserTasksHandler) Execute(ctx context.Context, req ListUserTasksRequest) ([]*ListUserTasksResponse, error) {
	tasks, err := h.repo.ListByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	responses := make([]*ListUserTasksResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = &ListUserTasksResponse{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			UserID:      t.UserID,
			Completed:   t.Completed,
		}
	}

	return responses, nil
}
