// Package app est la racine de composition : elle assemble repositories,
// handlers, controllers et middlewares en un routeur Echo prêt à servir.
package app

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	"github.com/fdanielsin/inbackend/internal/controllers/httpapi"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"
)

// maxRequestBody est la taille maximale d'un corps de requête (413 au-delà).
const maxRequestBody int64 = 1 << 20

// Option ajuste la configuration du routeur.
type Option func(*routerConfig)

type routerConfig struct {
	allowedOrigins []string
	log            *slog.Logger
	apiContract    []byte
}

// WithAllowedOrigins fixe les origines CORS autorisées ("*" par défaut).
func WithAllowedOrigins(origins []string) Option {
	return func(cfg *routerConfig) { cfg.allowedOrigins = origins }
}

// WithAPIDocs sert le contrat OpenAPI donné sur /openapi.yaml et l'interface
// Swagger UI sur /docs ; sans cette option, ces routes n'existent pas.
func WithAPIDocs(contract []byte) Option {
	return func(cfg *routerConfig) { cfg.apiContract = contract }
}

// WithLogger fixe le logger des controllers et du journal d'accès ; nil garde
// le logger par défaut de slog.
func WithLogger(log *slog.Logger) Option {
	return func(cfg *routerConfig) {
		if log != nil {
			cfg.log = log
		}
	}
}

// NewRouter construit le routeur de l'API sur la base fournie.
func NewRouter(db *gorm.DB, opts ...Option) *echo.Echo {
	cfg := routerConfig{allowedOrigins: []string{"*"}, log: slog.Default()}
	for _, opt := range opts {
		opt(&cfg)
	}
	log := cfg.log

	userRepo := pgstore.NewUserRepository(db)
	taskRepo := pgstore.NewTaskRepository(db)

	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)

	createTaskHandler := commands.NewCreateTaskHandler(taskRepo)
	updateTaskHandler := commands.NewUpdateTaskHandler(taskRepo)
	deleteTaskHandler := commands.NewDeleteTaskHandler(taskRepo)
	listUserTasksHandler := queries.NewListUserTasksHandler(taskRepo)
	getStatsHandler := queries.NewGetStatsHandler(userRepo, taskRepo)

	healthCtrl := httpapi.NewHealthController(log, databaseReadiness(db))
	userCtrl := httpapi.NewUserController(log, createUserHandler, getUserHandler, listUsersHandler)
	taskCtrl := httpapi.NewTaskController(log, createTaskHandler, updateTaskHandler, deleteTaskHandler, listUserTasksHandler)
	statsCtrl := httpapi.NewStatsController(log, getStatsHandler)

	e := echo.New()
	e.Logger = log

	e.HTTPErrorHandler = httpapi.NewErrorHandler(log)
	e.Use(httpapi.Language())
	e.Use(httpapi.RequestID(log))
	e.Use(httpapi.AccessLog(log))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  cfg.allowedOrigins,
		AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:  []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		ExposeHeaders: []string{echo.HeaderXRequestID},
	}))
	e.Use(httpapi.BodyLimit(maxRequestBody))

	e.GET("/health", healthCtrl.HealthCheck)
	e.GET("/ready", healthCtrl.Ready)
	e.GET("/api/v1/users", userCtrl.ListUsers)
	e.POST("/api/v1/users", userCtrl.CreateUser)
	e.GET("/api/v1/users/:id", userCtrl.GetUser)
	e.GET("/api/v1/users/:id/tasks", taskCtrl.ListUserTasks)
	e.POST("/api/v1/tasks", taskCtrl.CreateTask)
	e.PUT("/api/v1/tasks/:id", taskCtrl.UpdateTask)
	e.DELETE("/api/v1/tasks/:id", taskCtrl.DeleteTask)
	e.GET("/api/v1/stats", statsCtrl.GetStats)

	if cfg.apiContract != nil {
		docsCtrl := httpapi.NewDocsController(cfg.apiContract)
		e.GET("/openapi.yaml", docsCtrl.OpenAPI)
		e.GET("/docs", docsCtrl.SwaggerUI)
	}

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

// databaseReadiness vérifie la base à chaque appel (le pool peut se rétablir seul).
func databaseReadiness(db *gorm.DB) httpapi.ReadinessCheck {
	return func(ctx context.Context) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	}
}
