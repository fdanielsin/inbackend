// Command inbackend démarre l'API HTTP : il lit la configuration, ouvre la base et lance le routeur de internal/app.
package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"syscall"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/internal/controllers/httpapi"
	"github.com/fdanielsin/inbackend/internal/infrastructure/applog"
	"github.com/fdanielsin/inbackend/internal/infrastructure/pgsetup"
	"gorm.io/gorm"
)

// apiContract est le contrat OpenAPI : il valide toujours les corps de requête, et
// il est servi sur /openapi.yaml quand les docs sont actives.
//
//go:embed docs/specs/api/openapi.yaml
var apiContract []byte

func main() {
	log := applog.InitLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, log)
	stop()

	if err != nil {
		log.Error("Server failed", slog.Any("error", err))
		os.Exit(1)
	}
	log.Info("Server stopped")
}

func run(ctx context.Context, log *slog.Logger) error {
	origins, err := corsOrigins(os.Getenv("ENV"), os.Getenv("CORS_ALLOWED_ORIGINS"))
	if err != nil {
		return err
	}

	docsEnabled, err := apiDocsEnabled(os.Getenv("ENV"), os.Getenv("API_DOCS_ENABLED"))
	if err != nil {
		return err
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL not set")
	}

	validator, err := httpapi.NewRequestValidator(apiContract, app.ValidatedOperations...)
	if err != nil {
		return fmt.Errorf("failed to load API contract: %w", err)
	}

	db, err := pgsetup.InitDB(dsn, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer closeDB(db, log)

	opts := []app.Option{app.WithLogger(log), app.WithAllowedOrigins(origins), app.WithRequestValidator(validator)}
	if docsEnabled {
		opts = append(opts, app.WithAPIDocs(apiContract))
	}
	e := app.NewRouter(db, opts...)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	log.Info("Starting server", slog.String("port", port))
	return app.Serve(ctx, e, listener)
}

// corsOrigins lit CORS_ALLOWED_ORIGINS. Quand env vaut exactement "production", la
// liste doit nommer les origines autorisées : absente, vide ou contenant "*", elle
// empêche le démarrage. Toute origine mal formée l'empêche aussi (voir
// app.ValidateAllowedOrigins). Les messages ne reprennent pas la valeur reçue.
func corsOrigins(env, value string) ([]string, error) {
	origins := app.ParseAllowedOrigins(value)
	if env == "production" && slices.Contains(origins, "*") {
		return nil, errors.New("CORS_ALLOWED_ORIGINS must list the allowed origins when ENV=production (not empty, not *)")
	}
	if err := app.ValidateAllowedOrigins(origins); err != nil {
		return nil, err
	}
	return origins, nil
}

// apiDocsEnabled décide si /openapi.yaml et /docs existent : la valeur de
// API_DOCS_ENABLED si elle est donnée, sinon oui hors production.
func apiDocsEnabled(env, value string) (bool, error) {
	if value == "" {
		return env != "production", nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("API_DOCS_ENABLED must be true or false, got %q", value)
	}
	return enabled, nil
}

func closeDB(db *gorm.DB, log *slog.Logger) {
	sqlDB, err := db.DB()
	if err == nil {
		err = sqlDB.Close()
	}
	if err != nil {
		log.Warn("Failed to close the database", slog.Any("error", err))
	}
}
