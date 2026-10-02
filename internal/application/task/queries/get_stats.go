// Package queries regroupe les requêtes CQRS du contexte tâche.
package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
)

// GetStatsRequest est le DTO de la requête GetStats.
type GetStatsRequest struct{}

// GetStatsResponse est la réponse de la requête GetStats.
type GetStatsResponse struct {
	Users int64 `json:"users"`
	Tasks int64 `json:"tasks"`
}

// GetStatsHandler exécute la logique de récupération des statistiques.
type GetStatsHandler struct {
	userRepo users.Repository
	taskRepo tasks.Repository
}

// NewGetStatsHandler crée une nouvelle instance du handler.
func NewGetStatsHandler(userRepo users.Repository, taskRepo tasks.Repository) *GetStatsHandler {
	return &GetStatsHandler{
		userRepo: userRepo,
		taskRepo: taskRepo,
	}
}

// Execute exécute la requête GetStats.
func (h *GetStatsHandler) Execute(ctx context.Context, _ GetStatsRequest) (*GetStatsResponse, error) {
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	taskCount, err := h.taskRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	return &GetStatsResponse{
		Users: userCount,
		Tasks: taskCount,
	}, nil
}
