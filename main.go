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
	"strconv"
	"syscall"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"gorm.io/gorm"
)

// apiContract est le contrat OpenAPI, servi sur /openapi.yaml quand les docs sont actives.
//
//go:embed docs/specs/api/openapi.yaml
var apiContract []byte

func main() {
	log := logger.InitLogger()

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
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL not set")
	}

	docsEnabled, err := apiDocsEnabled(os.Getenv("ENV"), os.Getenv("API_DOCS_ENABLED"))
	if err != nil {
		return err
	}

	db, err := database.InitDB(dsn, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer closeDB(db, log)

	opts := []app.Option{app.WithLogger(log), app.WithAllowedOrigins(app.ParseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")))}
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
