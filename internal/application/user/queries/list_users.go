package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/application"
	"github.com/fdanielsin/inbackend/internal/domain/users"
)

// ListUsersRequest est le DTO de la requête ListUsers.
type ListUsersRequest struct{}

// ListUsersHandler exécute la logique de récupération de tous les utilisateurs.
type ListUsersHandler struct {
	repo users.Repository
}

// NewListUsersHandler crée une nouvelle instance du handler.
func NewListUsersHandler(repo users.Repository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

// Execute exécute la requête ListUsers.
func (h *ListUsersHandler) Execute(ctx context.Context, _ ListUsersRequest) ([]*application.UserResponse, error) {
	users, err := h.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*application.UserResponse, len(users))
	for i, u := range users {
		responses[i] = &application.UserResponse{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		}
	}

	return responses, nil
}
