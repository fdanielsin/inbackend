package main

import (
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type HealthData struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	ID      string `json:"id"`
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	e.GET("/health", healthHandler)
	e.GET("/api/v1/status", statusHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Service is healthy",
		Data: HealthData{
			Status:  "ok",
			Version: "1.0.0",
			ID:      uuid.New().String(),
		},
	})
}

func statusHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"timestamp": c.Request().Header.Get("Date"),
			"uptime":    "available",
		},
	})
}
