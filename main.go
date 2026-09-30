// Command inbackend démarre l'API HTTP : il assemble les dépendances et lance Echo.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	ctrlhttp "github.com/fdanielsin/inbackend/internal/controllers/http"
	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	log := logger.InitLogger()

	// Fatal vide les buffers de zap avant de quitter : aucun Sync différé n'est nécessaire.
	if err := run(log); err != nil {
		log.Fatal("Server failed", zap.Error(err))
	}
}

func run(log *zap.Logger) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL not set")
	}

	db, err := database.InitDB(dsn, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialiser les repositories PostgreSQL
	userRepo := postgres.NewUserRepository(db)
	taskRepo := postgres.NewTaskRepository(db)

	// Initialiser les handlers (commands et queries)
	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)

	createTaskHandler := commands.NewCreateTaskHandler(taskRepo)
	updateTaskHandler := commands.NewUpdateTaskHandler(taskRepo)
	deleteTaskHandler := commands.NewDeleteTaskHandler(taskRepo)
	listUserTasksHandler := queries.NewListUserTasksHandler(taskRepo)
	getStatsHandler := queries.NewGetStatsHandler(userRepo, taskRepo)

	// Initialiser les controllers
	healthCtrl := ctrlhttp.NewHealthController()
	userCtrl := ctrlhttp.NewUserController(createUserHandler, getUserHandler, listUsersHandler)
	taskCtrl := ctrlhttp.NewTaskController(createTaskHandler, updateTaskHandler, deleteTaskHandler, listUserTasksHandler)
	statsCtrl := ctrlhttp.NewStatsController(getStatsHandler)

	// Configurer Echo
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Enregistrer les routes
	e.GET("/health", healthCtrl.HealthCheck)
	e.GET("/api/v1/users", userCtrl.ListUsers)
	e.POST("/api/v1/users", userCtrl.CreateUser)
	e.GET("/api/v1/users/:id", userCtrl.GetUser)
	e.GET("/api/v1/users/:id/tasks", taskCtrl.ListUserTasks)
	e.POST("/api/v1/tasks", taskCtrl.CreateTask)
	e.PUT("/api/v1/tasks/:id", taskCtrl.UpdateTask)
	e.DELETE("/api/v1/tasks/:id", taskCtrl.DeleteTask)
	e.GET("/api/v1/stats", statsCtrl.GetStats)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info("Starting server", zap.String("port", port))
	return e.Start(":" + port)
}
