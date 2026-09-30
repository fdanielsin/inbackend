// Package queries regroupe les requêtes CQRS du contexte utilisateur.
package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// GetUserRequest est le DTO de la requête GetUser.
type GetUserRequest struct {
	ID uint
}

// GetUserResponse est la réponse de la requête GetUser.
type GetUserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetUserHandler exécute la logique de récupération d'un utilisateur.
type GetUserHandler struct {
	repo user.Repository
}

// NewGetUserHandler crée une nouvelle instance du handler.
func NewGetUserHandler(repo user.Repository) *GetUserHandler {
	return &GetUserHandler{repo: repo}
}

// Execute exécute la requête GetUser.
func (h *GetUserHandler) Execute(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
	u, err := h.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	return &GetUserResponse{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}
