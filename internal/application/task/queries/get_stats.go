package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
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
	userRepo user.Repository
	taskRepo task.Repository
}

// NewGetStatsHandler crée une nouvelle instance du handler.
func NewGetStatsHandler(userRepo user.Repository, taskRepo task.Repository) *GetStatsHandler {
	return &GetStatsHandler{
		userRepo: userRepo,
		taskRepo: taskRepo,
	}
}

// Execute exécute la requête GetStats.
func (h *GetStatsHandler) Execute(ctx context.Context, req GetStatsRequest) (*GetStatsResponse, error) {
	users, err := h.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	taskCount, err := h.taskRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	return &GetStatsResponse{
		Users: int64(len(users)),
		Tasks: taskCount,
	}, nil
}
