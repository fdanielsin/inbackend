package http

import (
	"log/slog"
	"net/http"

	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
)

// StatsController gère les routes de statistiques.
type StatsController struct {
	getStatsHandler *queries.GetStatsHandler
	log             *slog.Logger
}

// NewStatsController crée une nouvelle instance du controller.
func NewStatsController(log *slog.Logger, getStatsHandler *queries.GetStatsHandler) *StatsController {
	return &StatsController{
		getStatsHandler: getStatsHandler,
		log:             log,
	}
}

// GetStats gère GET /api/v1/stats.
func (sc *StatsController) GetStats(c echo.Context) error {
	log := requestLogger(c, sc.log)
	log.Info("Fetching statistics")

	resp, err := sc.getStatsHandler.Execute(c.Request().Context(), queries.GetStatsRequest{})
	if err != nil {
		return respondWithError(c, log, "Error fetching statistics", err)
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}
