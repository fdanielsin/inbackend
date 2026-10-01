// Package queries regroupe les requêtes CQRS du contexte utilisateur.
package queries

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/application"
	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// GetUserRequest est le DTO de la requête GetUser.
type GetUserRequest struct {
	ID uint
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
func (h *GetUserHandler) Execute(ctx context.Context, req GetUserRequest) (*application.UserResponse, error) {
	u, err := h.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	return &application.UserResponse{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}
