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
	CodeInvalidBody         = "INVALID_BODY"
	CodeInvalidID           = "INVALID_ID"
	CodeValidationFailed    = "VALIDATION_FAILED"
	CodeUserNotFound        = "USER_NOT_FOUND"
	CodeTaskNotFound        = "TASK_NOT_FOUND"
	CodeEmailTaken          = "EMAIL_TAKEN"
	CodeRouteNotFound       = "ROUTE_NOT_FOUND"
	CodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	CodePayloadTooLarge     = "PAYLOAD_TOO_LARGE"
	CodeDatabaseUnavailable = "DATABASE_UNAVAILABLE"
	CodeInternalError       = "INTERNAL_ERROR"
)

// APIError est la forme publique d'une erreur : statut HTTP, code stable et clé
// du message du catalogue.
type APIError struct {
	Status int
	Code   string
	Key    MessageKey
}

// Message retourne le texte de l'erreur dans la langue donnée.
func (e APIError) Message(lang Lang) string {
	return Message(lang, e.Key)
}

// internalError est la réponse de toute erreur non listée : aucun détail technique n'est exposé.
var internalError = APIError{http.StatusInternalServerError, CodeInternalError, MsgInternalError}

// domainErrors est l'unique table erreur domaine -> réponse publique.
var domainErrors = []struct {
	err error
	api APIError
}{
	{user.ErrUserNotFound, APIError{http.StatusNotFound, CodeUserNotFound, MsgUserNotFound}},
	{user.ErrUserNameEmpty, APIError{http.StatusBadRequest, CodeValidationFailed, MsgUserNameEmpty}},
	{user.ErrUserEmailEmpty, APIError{http.StatusBadRequest, CodeValidationFailed, MsgUserEmailEmpty}},
	{user.ErrUserEmailInvalid, APIError{http.StatusBadRequest, CodeValidationFailed, MsgUserEmailInvalid}},
	{user.ErrUserEmailTaken, APIError{http.StatusConflict, CodeEmailTaken, MsgEmailTaken}},
	{task.ErrTaskNotFound, APIError{http.StatusNotFound, CodeTaskNotFound, MsgTaskNotFound}},
	{task.ErrTaskTitleEmpty, APIError{http.StatusBadRequest, CodeValidationFailed, MsgTaskTitleEmpty}},
	{task.ErrTaskUserIDInvalid, APIError{http.StatusBadRequest, CodeValidationFailed, MsgTaskUserIDInvalid}},
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
