package http

import (
	"log/slog"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/logger"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// HealthController gère les routes de santé.
type HealthController struct {
	log *slog.Logger
}

// NewHealthController crée une nouvelle instance du controller.
func NewHealthController() *HealthController {
	return &HealthController{
		log: logger.GetSlog(),
	}
}

// HealthCheck gère GET /health.
func (hc *HealthController) HealthCheck(c echo.Context) error {
	hc.log.Debug("Health check requested")

	return c.JSON(http.StatusOK, shared.SuccessResponse(map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"id":      uuid.New().String(),
	}, "Service is healthy"))
}
