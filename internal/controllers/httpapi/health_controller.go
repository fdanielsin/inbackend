// Package httpapi expose les controllers HTTP Echo de l'API.
package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// readinessTimeout borne la vérification de la base : inférieur au délai d'une
// sonde habituelle (3 s), pour répondre 503 plutôt que laisser la sonde expirer.
const readinessTimeout = 2 * time.Second

// ReadinessCheck vérifie que les dépendances du service répondent.
type ReadinessCheck func(ctx context.Context) error

// HealthController gère les routes de santé.
type HealthController struct {
	log   *slog.Logger
	check ReadinessCheck
}

// NewHealthController crée une nouvelle instance du controller.
func NewHealthController(log *slog.Logger, check ReadinessCheck) *HealthController {
	return &HealthController{
		log:   log,
		check: check,
	}
}

// HealthCheck gère GET /health : vivacité, sans dépendance à la base.
func (hc *HealthController) HealthCheck(c *echo.Context) error {
	log := requestLogger(c, hc.log)
	log.Debug("Health check requested")
	forbidCaching(c)

	return c.JSON(http.StatusOK, shared.SuccessResponse(map[string]any{
		"status":  "ok",
		"version": "1.1.0",
		"id":      uuid.New().String(),
	}, shared.Message(requestLanguage(c), shared.MsgServiceHealthy)))
}

// Ready gère GET /ready : disponibilité, la base doit répondre.
func (hc *HealthController) Ready(c *echo.Context) error {
	log := requestLogger(c, hc.log)
	forbidCaching(c)

	ctx, cancel := context.WithTimeout(c.Request().Context(), readinessTimeout)
	defer cancel()
	if err := hc.check(ctx); err != nil {
		log.Error("Readiness check failed",
			slog.String("error_code", shared.CodeDatabaseUnavailable),
			slog.String("error_type", fmt.Sprintf("%T", err)))
		return c.JSON(http.StatusServiceUnavailable,
			shared.ErrorResponse(shared.CodeDatabaseUnavailable, shared.Message(requestLanguage(c), shared.MsgDatabaseDown)))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(map[string]string{"status": "ready"}, shared.Message(requestLanguage(c), shared.MsgServiceReady)))
}

// forbidCaching empêche tout proxy ou client de garder l'état d'une sonde.
func forbidCaching(c *echo.Context) {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
}
