package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
)

// DeleteTaskRequest est le DTO de la commande DeleteTask.
type DeleteTaskRequest struct {
	ID uint
}

// DeleteTaskHandler exécute la logique de suppression de tâche.
type DeleteTaskHandler struct {
	repo tasks.Repository
}

// NewDeleteTaskHandler crée une nouvelle instance du handler.
func NewDeleteTaskHandler(repo tasks.Repository) *DeleteTaskHandler {
	return &DeleteTaskHandler{repo: repo}
}

// Execute exécute la commande DeleteTask.
func (h *DeleteTaskHandler) Execute(ctx context.Context, req DeleteTaskRequest) error {
	return h.repo.Delete(ctx, req.ID)
}
