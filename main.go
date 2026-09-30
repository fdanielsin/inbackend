// Command inbackend démarre l'API HTTP : il lit la configuration, ouvre la base et lance le routeur de internal/app.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
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

	e := app.NewRouter(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info("Starting server", zap.String("port", port))
	return e.Start(":" + port)
}
