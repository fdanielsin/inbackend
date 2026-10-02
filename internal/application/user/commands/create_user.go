// Package commands regroupe les commandes CQRS du contexte utilisateur.
package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/application"
	"github.com/fdanielsin/inbackend/internal/domain/users"
)

// CreateUserRequest est le DTO de la commande CreateUser.
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserHandler exécute la logique de création d'utilisateur.
type CreateUserHandler struct {
	repo users.Repository
}

// NewCreateUserHandler crée une nouvelle instance du handler.
func NewCreateUserHandler(repo users.Repository) *CreateUserHandler {
	return &CreateUserHandler{repo: repo}
}

// Execute exécute la commande CreateUser.
func (h *CreateUserHandler) Execute(ctx context.Context, req CreateUserRequest) (*application.UserResponse, error) {
	u, err := users.NewUser(0, req.Name, req.Email)
	if err != nil {
		return nil, err
	}

	id, err := h.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	return &application.UserResponse{
		ID:    id,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}
