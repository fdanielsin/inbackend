// Command inbackend démarre l'API HTTP : il lit la configuration, ouvre la base et lance le routeur de internal/app.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"gorm.io/gorm"
)

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

	db, err := database.InitDB(dsn, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer closeDB(db, log)

	e := app.NewRouter(db, app.WithLogger(log), app.WithAllowedOrigins(app.ParseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))))

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

func closeDB(db *gorm.DB, log *slog.Logger) {
	sqlDB, err := db.DB()
	if err == nil {
		err = sqlDB.Close()
	}
	if err != nil {
		log.Warn("Failed to close the database", slog.Any("error", err))
	}
}
