package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/application"
	"github.com/fdanielsin/inbackend/internal/domain/users"
)

// UpdateUserRequest est le DTO de la commande UpdateUser : seul le nom change.
// ID vient du chemin, jamais du corps (json:"-").
type UpdateUserRequest struct {
	ID   uint   `json:"-"`
	Name string `json:"name"`
}

// UpdateUserHandler exécute la logique de renommage d'un utilisateur.
type UpdateUserHandler struct {
	repo users.Repository
}

// NewUpdateUserHandler crée une nouvelle instance du handler.
func NewUpdateUserHandler(repo users.Repository) *UpdateUserHandler {
	return &UpdateUserHandler{repo: repo}
}

// Execute exécute la commande UpdateUser.
func (h *UpdateUserHandler) Execute(ctx context.Context, req UpdateUserRequest) (*application.UserResponse, error) {
	name, err := users.NormalizeName(req.Name)
	if err != nil {
		return nil, err
	}

	u, err := h.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	u.Name = name

	if err := h.repo.Update(ctx, u); err != nil {
		return nil, err
	}

	return &application.UserResponse{ID: u.ID, Name: u.Name, Email: u.Email}, nil
}
