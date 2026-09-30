package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
)

// respondWithError traduit une erreur du domaine ou technique en réponse
// publique (statut, code stable, message) et la journalise : erreur pour un
// 5xx, information pour un 404, avertissement sinon.
// Le texte de l'erreur n'est jamais journalisé : celui de PostgreSQL ou de
// GORM peut contenir des valeurs utilisateur (« Key (email)=(…) »). Seuls le
// code public et, pour une erreur inconnue, son type permettent de diagnostiquer.
func respondWithError(c echo.Context, log *slog.Logger, action string, err error, attrs ...any) error {
	apiErr := shared.Classify(err)

	level := slog.LevelWarn
	switch {
	case apiErr.Status >= http.StatusInternalServerError:
		level = slog.LevelError
		attrs = append(attrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	case apiErr.Status == http.StatusNotFound:
		level = slog.LevelInfo
	}
	log.Log(c.Request().Context(), level, action, append(attrs, slog.String("error_code", apiErr.Code))...)

	return c.JSON(apiErr.Status, shared.ErrorResponse(apiErr.Code, apiErr.Message))
}

func respondInvalidBody(c echo.Context, log *slog.Logger) error {
	log.Warn("Invalid request body")
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeInvalidBody, "Invalid request body"))
}

func respondInvalidID(c echo.Context, log *slog.Logger, resource, rawID string) error {
	log.Warn("Invalid "+resource+" ID", slog.String("id", rawID))
	return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.CodeInvalidID, "Invalid "+resource+" ID"))
}
