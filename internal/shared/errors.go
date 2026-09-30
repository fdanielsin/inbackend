// Package shared contient les erreurs HTTP et l'enveloppe de réponse communes.
package shared

import (
	"errors"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// MapErrorToMessage mappe une erreur domaine à un message HTTP.
func MapErrorToMessage(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, user.ErrUserNotFound):
		return "User not found"
	case errors.Is(err, user.ErrUserNameEmpty):
		return "User name cannot be empty"
	case errors.Is(err, user.ErrUserEmailEmpty):
		return "User email cannot be empty"
	case errors.Is(err, task.ErrTaskNotFound):
		return "Task not found"
	case errors.Is(err, task.ErrTaskTitleEmpty):
		return "Task title cannot be empty"
	case errors.Is(err, task.ErrTaskUserIDInvalid):
		return "Task user_id cannot be empty"
	default:
		return "Internal server error"
	}
}

// IsNotFoundError retourne true si l'erreur est une erreur "not found".
func IsNotFoundError(err error) bool {
	return errors.Is(err, user.ErrUserNotFound) || errors.Is(err, task.ErrTaskNotFound)
}

// IsValidationError retourne true si l'erreur est une erreur de validation.
func IsValidationError(err error) bool {
	switch {
	case errors.Is(err, user.ErrUserNameEmpty),
		errors.Is(err, user.ErrUserEmailEmpty),
		errors.Is(err, task.ErrTaskTitleEmpty),
		errors.Is(err, task.ErrTaskUserIDInvalid):
		return true
	default:
		return false
	}
}
