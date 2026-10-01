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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	log, slogLogger, err := logger.InitLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err = run(ctx, log, slogLogger)
	stop()

	// Fatal vide les buffers de zap avant de quitter : aucun Sync différé n'est nécessaire.
	if err != nil {
		log.Fatal("Server failed", zap.Error(err))
	}
	log.Info("Server stopped")
}

func run(ctx context.Context, log *zap.Logger, slogLogger *slog.Logger) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL not set")
	}

	db, err := database.InitDB(dsn, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer closeDB(db, log)

	e := app.NewRouter(db, app.WithLogger(slogLogger), app.WithAllowedOrigins(app.ParseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	log.Info("Starting server", zap.String("port", port))
	return app.Serve(ctx, e, listener)
}

func closeDB(db *gorm.DB, log *zap.Logger) {
	sqlDB, err := db.DB()
	if err == nil {
		err = sqlDB.Close()
	}
	if err != nil {
		log.Warn("Failed to close the database", zap.Error(err))
	}
}
