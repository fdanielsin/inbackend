package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/task"
)

// UpdateTaskRequest est le DTO de la commande UpdateTask.
type UpdateTaskRequest struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// UpdateTaskResponse est la réponse de la commande UpdateTask.
type UpdateTaskResponse struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// UpdateTaskHandler exécute la logique de mise à jour de tâche.
type UpdateTaskHandler struct {
	repo task.Repository
}

// NewUpdateTaskHandler crée une nouvelle instance du handler.
func NewUpdateTaskHandler(repo task.Repository) *UpdateTaskHandler {
	return &UpdateTaskHandler{repo: repo}
}

// Execute exécute la commande UpdateTask.
func (h *UpdateTaskHandler) Execute(ctx context.Context, req UpdateTaskRequest) (*UpdateTaskResponse, error) {
	// Récupérer la tâche existante
	t, err := h.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	// Mettre à jour les propriétés
	t.Title = req.Title
	t.Description = req.Description
	t.Completed = req.Completed

	// Persister
	if err := h.repo.Update(ctx, t); err != nil {
		return nil, err
	}

	return &UpdateTaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Completed:   t.Completed,
	}, nil
}
