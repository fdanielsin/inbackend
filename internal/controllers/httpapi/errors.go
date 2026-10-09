package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
)

// logLevelForStatus : erreur pour un 5xx, avertissement pour un 4xx, sinon information.
func logLevelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// respondWithError traduit une erreur du domaine ou technique en réponse
// publique (statut, code stable, message) et la journalise : erreur pour un
// 5xx, information pour un 404, avertissement sinon.
// Le texte de l'erreur n'est jamais journalisé : celui de PostgreSQL ou de
// GORM peut contenir des valeurs utilisateur (« Key (email)=(…) »). Seuls le
// code public et, pour une erreur inconnue, son type permettent de diagnostiquer.
func respondWithError(c *echo.Context, log *slog.Logger, action string, err error, attrs ...any) error {
	apiErr := shared.Classify(err)

	level := logLevelForStatus(apiErr.Status)
	if apiErr.Status == http.StatusNotFound {
		level = slog.LevelInfo
	}
	if apiErr.Status >= http.StatusInternalServerError {
		attrs = append(attrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	}
	log.Log(c.Request().Context(), level, action, append(attrs, slog.String("error_code", apiErr.Code))...)

	return c.JSON(apiErr.Status, shared.ErrorResponse(apiErr.Code, apiErr.Message(requestLanguage(c))))
}

// respondInvalidBody répond 400 pour un corps illisible, ou 413 quand c'est la
// limite de taille qui a interrompu la lecture (Bind enveloppe alors l'erreur
// de lecture dans un 400).
func respondInvalidBody(c *echo.Context, log *slog.Logger, err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return respondPayloadTooLarge(c, log)
	}
	log.Warn("Invalid request body")
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeInvalidBody, shared.Message(requestLanguage(c), shared.MsgInvalidBody)))
}

func respondPayloadTooLarge(c *echo.Context, log *slog.Logger) error {
	log.Warn("Request body too large", slog.String("error_code", shared.CodePayloadTooLarge))
	return c.JSON(http.StatusRequestEntityTooLarge,
		shared.ErrorResponse(shared.CodePayloadTooLarge, shared.Message(requestLanguage(c), shared.MsgPayloadTooLarge)))
}

func respondInvalidID(c *echo.Context, log *slog.Logger, key shared.MessageKey, rawID string) error {
	log.Warn(shared.Message(shared.LangEnglish, key), slog.String("id", rawID))
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeInvalidID, shared.Message(requestLanguage(c), key)))
}

// respondInvalidCompletedFilter répond 400 quand le paramètre « completed » n'est ni « true » ni « false ».
// La valeur reçue n'est pas journalisée.
func respondInvalidCompletedFilter(c *echo.Context, log *slog.Logger) error {
	log.Warn("Invalid completed filter", slog.String("error_code", shared.CodeValidationFailed))
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeValidationFailed, shared.Message(requestLanguage(c), shared.MsgTaskCompletedFilter)))
}

// NewErrorHandler renvoie le HTTPErrorHandler d'Echo : les erreurs qu'Echo
// produit lui-même (route inconnue, méthode non supportée, corps trop gros,
// panique récupérée) sortent dans la même enveloppe que celles des controllers.
func NewErrorHandler(base *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if response, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil && response.Committed {
			return
		}
		log := requestLogger(c, base)

		status, code, key := http.StatusInternalServerError, shared.CodeInternalError, shared.MsgInternalError
		var statusErr echo.HTTPStatusCoder
		if errors.As(err, &statusErr) {
			httpStatus := statusErr.StatusCode()
			switch httpStatus {
			case http.StatusNotFound:
				status, code, key = httpStatus, shared.CodeRouteNotFound, shared.MsgRouteNotFound
			case http.StatusMethodNotAllowed:
				status, code, key = httpStatus, shared.CodeMethodNotAllowed, shared.MsgMethodNotAllowed
			case http.StatusRequestEntityTooLarge:
				status, code, key = httpStatus, shared.CodePayloadTooLarge, shared.MsgPayloadTooLarge
			default:
				if httpStatus < http.StatusInternalServerError {
					status, code, key = httpStatus, shared.CodeInvalidBody, shared.MsgInvalidRequest
				}
			}
		}

		attrs := []any{slog.String("error_code", code)}
		if status >= http.StatusInternalServerError {
			attrs = append(attrs, slog.String("error_type", fmt.Sprintf("%T", err)))
		}
		log.Log(c.Request().Context(), logLevelForStatus(status), "Request failed", attrs...)

		if sendErr := c.JSON(status, shared.ErrorResponse(code, shared.Message(requestLanguage(c), key))); sendErr != nil {
			log.Error("Failed to write error response", slog.String("error_type", fmt.Sprintf("%T", sendErr)))
		}
	}
}
