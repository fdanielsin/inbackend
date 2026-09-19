package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// ListUsersRequest est le DTO de la requête ListUsers.
type ListUsersRequest struct{}

// ListUsersResponse est la réponse de la requête ListUsers.
type ListUsersResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ListUsersHandler exécute la logique de récupération de tous les utilisateurs.
type ListUsersHandler struct {
	repo user.Repository
}

// NewListUsersHandler crée une nouvelle instance du handler.
func NewListUsersHandler(repo user.Repository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

// Execute exécute la requête ListUsers.
func (h *ListUsersHandler) Execute(ctx context.Context, req ListUsersRequest) ([]*ListUsersResponse, error) {
	users, err := h.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*ListUsersResponse, len(users))
	for i, u := range users {
		responses[i] = &ListUsersResponse{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		}
	}

	return responses, nil
}
