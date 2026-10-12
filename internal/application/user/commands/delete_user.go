package commands

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/users"
)

// DeleteUserRequest est le DTO de la commande DeleteUser.
type DeleteUserRequest struct {
	ID uint
}

// DeleteUserHandler exécute la logique de suppression d'un utilisateur et de ses tâches.
type DeleteUserHandler struct {
	repo users.Repository
}

// NewDeleteUserHandler crée une nouvelle instance du handler.
func NewDeleteUserHandler(repo users.Repository) *DeleteUserHandler {
	return &DeleteUserHandler{repo: repo}
}

// Execute exécute la commande DeleteUser.
func (h *DeleteUserHandler) Execute(ctx context.Context, req DeleteUserRequest) error {
	return h.repo.Delete(ctx, req.ID)
}
