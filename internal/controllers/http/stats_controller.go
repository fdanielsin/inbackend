package http

import (
	"log/slog"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	"github.com/fdanielsin/inbackend/internal/logger"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
)

// StatsController gère les routes de statistiques.
type StatsController struct {
	getStatsHandler *queries.GetStatsHandler
	log             *slog.Logger
}

// NewStatsController crée une nouvelle instance du controller.
func NewStatsController(getStatsHandler *queries.GetStatsHandler) *StatsController {
	return &StatsController{
		getStatsHandler: getStatsHandler,
		log:             logger.GetSlog(),
	}
}

// GetStats gère GET /api/v1/stats.
func (sc *StatsController) GetStats(c echo.Context) error {
	sc.log.Info("Fetching statistics")

	resp, err := sc.getStatsHandler.Execute(c.Request().Context(), queries.GetStatsRequest{})
	if err != nil {
		sc.log.Error("Error fetching statistics", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}
