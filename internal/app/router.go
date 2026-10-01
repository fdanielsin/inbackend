// Package app est la racine de composition : elle assemble repositories,
// handlers, controllers et middlewares en un routeur Echo prêt à servir.
package app

import (
	"net/http"
	"strings"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	ctrlhttp "github.com/fdanielsin/inbackend/internal/controllers/http"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// maxRequestBody est la taille maximale d'un corps de requête (413 au-delà).
const maxRequestBody int64 = 1 << 20

// Option ajuste la configuration du routeur.
type Option func(*routerConfig)

type routerConfig struct {
	allowedOrigins []string
}

// WithAllowedOrigins fixe les origines CORS autorisées ("*" par défaut).
func WithAllowedOrigins(origins []string) Option {
	return func(cfg *routerConfig) { cfg.allowedOrigins = origins }
}

// NewRouter construit le routeur de l'API sur la base fournie.
func NewRouter(db *gorm.DB, opts ...Option) *echo.Echo {
	cfg := routerConfig{allowedOrigins: []string{"*"}}
	for _, opt := range opts {
		opt(&cfg)
	}
	log := logger.GetSlog()

	userRepo := postgres.NewUserRepository(db)
	taskRepo := postgres.NewTaskRepository(db)

	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)

	createTaskHandler := commands.NewCreateTaskHandler(taskRepo)
	updateTaskHandler := commands.NewUpdateTaskHandler(taskRepo)
	deleteTaskHandler := commands.NewDeleteTaskHandler(taskRepo)
	listUserTasksHandler := queries.NewListUserTasksHandler(taskRepo)
	getStatsHandler := queries.NewGetStatsHandler(userRepo, taskRepo)

	healthCtrl := ctrlhttp.NewHealthController()
	userCtrl := ctrlhttp.NewUserController(createUserHandler, getUserHandler, listUsersHandler)
	taskCtrl := ctrlhttp.NewTaskController(createTaskHandler, updateTaskHandler, deleteTaskHandler, listUserTasksHandler)
	statsCtrl := ctrlhttp.NewStatsController(getStatsHandler)

	e := echo.New()

	e.HTTPErrorHandler = ctrlhttp.NewErrorHandler(log)
	e.Use(ctrlhttp.RequestID(log))
	e.Use(ctrlhttp.AccessLog(log))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  cfg.allowedOrigins,
		AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:  []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		ExposeHeaders: []string{echo.HeaderXRequestID},
	}))
	e.Use(ctrlhttp.BodyLimit(maxRequestBody))

	e.GET("/health", healthCtrl.HealthCheck)
	e.GET("/api/v1/users", userCtrl.ListUsers)
	e.POST("/api/v1/users", userCtrl.CreateUser)
	e.GET("/api/v1/users/:id", userCtrl.GetUser)
	e.GET("/api/v1/users/:id/tasks", taskCtrl.ListUserTasks)
	e.POST("/api/v1/tasks", taskCtrl.CreateTask)
	e.PUT("/api/v1/tasks/:id", taskCtrl.UpdateTask)
	e.DELETE("/api/v1/tasks/:id", taskCtrl.DeleteTask)
	e.GET("/api/v1/stats", statsCtrl.GetStats)

	return e
}

// ParseAllowedOrigins lit CORS_ALLOWED_ORIGINS : origines séparées par des
// virgules, espaces et éléments vides ignorés ; vide donne ["*"].
func ParseAllowedOrigins(value string) []string {
	var origins []string
	for _, origin := range strings.Split(value, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}
