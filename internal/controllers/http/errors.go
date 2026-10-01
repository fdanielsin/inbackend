package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
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
func respondWithError(c echo.Context, log *slog.Logger, action string, err error, attrs ...any) error {
	apiErr := shared.Classify(err)

	level := logLevelForStatus(apiErr.Status)
	if apiErr.Status == http.StatusNotFound {
		level = slog.LevelInfo
	}
	if apiErr.Status >= http.StatusInternalServerError {
		attrs = append(attrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	}
	log.Log(c.Request().Context(), level, action, append(attrs, slog.String("error_code", apiErr.Code))...)

	return c.JSON(apiErr.Status, shared.ErrorResponse(apiErr.Code, apiErr.Message))
}

// respondInvalidBody répond 400 pour un corps illisible, ou 413 quand c'est la
// limite de taille qui a interrompu la lecture (Bind enveloppe alors l'erreur
// de lecture dans un 400).
func respondInvalidBody(c echo.Context, log *slog.Logger, err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return respondPayloadTooLarge(c, log)
	}
	log.Warn("Invalid request body")
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeInvalidBody, "Invalid request body"))
}

func respondPayloadTooLarge(c echo.Context, log *slog.Logger) error {
	log.Warn("Request body too large", slog.String("error_code", shared.CodePayloadTooLarge))
	return c.JSON(http.StatusRequestEntityTooLarge,
		shared.ErrorResponse(shared.CodePayloadTooLarge, "Request body too large"))
}

func respondInvalidID(c echo.Context, log *slog.Logger, resource, rawID string) error {
	log.Warn("Invalid "+resource+" ID", slog.String("id", rawID))
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeInvalidID, "Invalid "+resource+" ID"))
}

// NewErrorHandler renvoie le HTTPErrorHandler d'Echo : les erreurs qu'Echo
// produit lui-même (route inconnue, méthode non supportée, corps trop gros,
// panique récupérée) sortent dans la même enveloppe que celles des controllers.
func NewErrorHandler(base *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		log := requestLogger(c, base)

		status, code, message := http.StatusInternalServerError, shared.CodeInternalError, "Internal server error"
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			switch httpErr.Code {
			case http.StatusNotFound:
				status, code, message = httpErr.Code, shared.CodeRouteNotFound, "Route not found"
			case http.StatusMethodNotAllowed:
				status, code, message = httpErr.Code, shared.CodeMethodNotAllowed, "Method not allowed"
			case http.StatusRequestEntityTooLarge:
				status, code, message = httpErr.Code, shared.CodePayloadTooLarge, "Request body too large"
			default:
				if httpErr.Code < http.StatusInternalServerError {
					status, code, message = httpErr.Code, shared.CodeInvalidBody, "Invalid request"
				}
			}
		}

		attrs := []any{slog.String("error_code", code)}
		if status >= http.StatusInternalServerError {
			attrs = append(attrs, slog.String("error_type", fmt.Sprintf("%T", err)))
		}
		log.Log(c.Request().Context(), logLevelForStatus(status), "Request failed", attrs...)

		var sendErr error
		if c.Request().Method == http.MethodHead {
			sendErr = c.NoContent(status)
		} else {
			sendErr = c.JSON(status, shared.ErrorResponse(code, message))
		}
		if sendErr != nil {
			log.Error("Failed to write error response", slog.String("error_type", fmt.Sprintf("%T", sendErr)))
		}
	}
}
