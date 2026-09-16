package main

import (
	"net/http"
	"os"

	"github.com/fdanielsin/inbackend/internal/db"
	"github.com/fdanielsin/inbackend/internal/handlers"
	"github.com/fdanielsin/inbackend/internal/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	log := logger.InitLogger()
	defer log.Sync()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	database, err := db.InitDB(dsn, log)
	if err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	h := &handlers.Handler{DB: database}

	e.GET("/health", h.HealthCheck)
	e.GET("/api/v1/users", h.GetUsers)
	e.POST("/api/v1/users", h.CreateUser)
	e.GET("/api/v1/users/:id", h.GetUserByID)
	e.GET("/api/v1/users/:id/tasks", h.GetUserTasks)
	e.POST("/api/v1/tasks", h.CreateTask)
	e.PUT("/api/v1/tasks/:id", h.UpdateTask)
	e.DELETE("/api/v1/tasks/:id", h.DeleteTask)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info("Starting server", zap.String("port", port))
	e.Logger.Fatal(e.Start(":" + port))
}
