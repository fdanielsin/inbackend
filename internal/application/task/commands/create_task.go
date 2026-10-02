// Package commands regroupe les commandes CQRS du contexte tâche.
package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
)

// CreateTaskRequest est le DTO de la commande CreateTask.
type CreateTaskRequest struct {
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// CreateTaskResponse est la réponse de la commande CreateTask.
type CreateTaskResponse struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// CreateTaskHandler exécute la logique de création de tâche.
type CreateTaskHandler struct {
	repo tasks.Repository
}

// NewCreateTaskHandler crée une nouvelle instance du handler.
func NewCreateTaskHandler(repo tasks.Repository) *CreateTaskHandler {
	return &CreateTaskHandler{repo: repo}
}

// Execute exécute la commande CreateTask.
func (h *CreateTaskHandler) Execute(ctx context.Context, req CreateTaskRequest) (*CreateTaskResponse, error) {
	t, err := tasks.NewTask(0, req.UserID, req.Title, req.Description, req.Completed)
	if err != nil {
		return nil, err
	}

	id, err := h.repo.Create(ctx, t)
	if err != nil {
		return nil, err
	}

	return &CreateTaskResponse{
		ID:          id,
		UserID:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		Completed:   t.Completed,
	}, nil
}
