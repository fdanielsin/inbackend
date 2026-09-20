package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/task"
)

// DeleteTaskRequest est le DTO de la commande DeleteTask.
type DeleteTaskRequest struct {
	ID uint
}

// DeleteTaskHandler exécute la logique de suppression de tâche.
type DeleteTaskHandler struct {
	repo task.Repository
}

// NewDeleteTaskHandler crée une nouvelle instance du handler.
func NewDeleteTaskHandler(repo task.Repository) *DeleteTaskHandler {
	return &DeleteTaskHandler{repo: repo}
}

// Execute exécute la commande DeleteTask.
func (h *DeleteTaskHandler) Execute(ctx context.Context, req DeleteTaskRequest) error {
	return h.repo.Delete(ctx, req.ID)
}
