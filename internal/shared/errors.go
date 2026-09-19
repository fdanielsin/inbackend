package shared

import (
	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// MapErrorToMessage mappe une erreur domaine à un message HTTP.
func MapErrorToMessage(err error) string {
	if err == nil {
		return ""
	}

	switch err {
	case user.ErrUserNotFound:
		return "User not found"
	case user.ErrUserNameEmpty:
		return "User name cannot be empty"
	case user.ErrUserEmailEmpty:
		return "User email cannot be empty"
	case task.ErrTaskNotFound:
		return "Task not found"
	case task.ErrTaskTitleEmpty:
		return "Task title cannot be empty"
	case task.ErrTaskUserIDInvalid:
		return "Task user_id cannot be empty"
	default:
		return err.Error()
	}
}

// IsNotFoundError retourne true si l'erreur est une erreur "not found".
func IsNotFoundError(err error) bool {
	return err == user.ErrUserNotFound || err == task.ErrTaskNotFound
}

// IsValidationError retourne true si l'erreur est une erreur de validation.
func IsValidationError(err error) bool {
	switch err {
	case user.ErrUserNameEmpty, user.ErrUserEmailEmpty,
		task.ErrTaskTitleEmpty, task.ErrTaskUserIDInvalid:
		return true
	default:
		return false
	}
}
