package main

import (
	"net/http"
	"os"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	ctrlhttp "github.com/fdanielsin/inbackend/internal/controllers/http"
	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
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

	database, err := database.InitDB(dsn, log)
	if err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialiser les repositories PostgreSQL
	userRepo := postgres.NewUserRepository(database)
	taskRepo := postgres.NewTaskRepository(database)

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
	e.Logger.Fatal(e.Start(":" + port))
}
