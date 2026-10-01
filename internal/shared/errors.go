// Package shared contient les erreurs HTTP et l'enveloppe de réponse communes.
package shared

import (
	"errors"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// Codes d'erreur stables exposés dans le champ error_code (voir openapi.yaml).
const (
	CodeInvalidBody      = "INVALID_BODY"
	CodeInvalidID        = "INVALID_ID"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeUserNotFound     = "USER_NOT_FOUND"
	CodeTaskNotFound     = "TASK_NOT_FOUND"
	CodeEmailTaken       = "EMAIL_TAKEN"
	CodeRouteNotFound    = "ROUTE_NOT_FOUND"
	CodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	CodePayloadTooLarge  = "PAYLOAD_TOO_LARGE"
	CodeInternalError    = "INTERNAL_ERROR"
)

// APIError est la forme publique d'une erreur : statut HTTP, code stable et message.
type APIError struct {
	Status  int
	Code    string
	Message string
}

// internalError est la réponse de toute erreur non listée : aucun détail technique n'est exposé.
var internalError = APIError{http.StatusInternalServerError, CodeInternalError, "Internal server error"}

// domainErrors est l'unique table erreur domaine -> réponse publique.
var domainErrors = []struct {
	err error
	api APIError
}{
	{user.ErrUserNotFound, APIError{http.StatusNotFound, CodeUserNotFound, "User not found"}},
	{user.ErrUserNameEmpty, APIError{http.StatusBadRequest, CodeValidationFailed, "User name cannot be empty"}},
	{user.ErrUserEmailEmpty, APIError{http.StatusBadRequest, CodeValidationFailed, "User email cannot be empty"}},
	{user.ErrUserEmailInvalid, APIError{http.StatusBadRequest, CodeValidationFailed, "User email is invalid"}},
	{user.ErrUserEmailTaken, APIError{http.StatusConflict, CodeEmailTaken, "Email already in use"}},
	{task.ErrTaskNotFound, APIError{http.StatusNotFound, CodeTaskNotFound, "Task not found"}},
	{task.ErrTaskTitleEmpty, APIError{http.StatusBadRequest, CodeValidationFailed, "Task title cannot be empty"}},
	{task.ErrTaskUserIDInvalid, APIError{http.StatusBadRequest, CodeValidationFailed, "Task user_id cannot be empty"}},
}

// Classify retourne la réponse publique d'une erreur ; toute erreur inconnue est une erreur interne.
func Classify(err error) APIError {
	for _, entry := range domainErrors {
		if errors.Is(err, entry.err) {
			return entry.api
		}
	}
	return internalError
}

// MapErrorToMessage retourne le message public d'une erreur ("" si err est nil).
func MapErrorToMessage(err error) string {
	if err == nil {
		return ""
	}
	return Classify(err).Message
}

// IsNotFoundError retourne true si l'erreur est une erreur "not found".
func IsNotFoundError(err error) bool {
	return err != nil && Classify(err).Status == http.StatusNotFound
}

// IsValidationError retourne true si l'erreur est une erreur de validation.
func IsValidationError(err error) bool {
	return err != nil && Classify(err).Code == CodeValidationFailed
}
