package logger

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/zap"
)

// Global logger instances
var SlogLog *slog.Logger

// InitLogger initialise les deux loggers : zap et slog.
// slog délègue vers zap en arrière-plan.
func InitLogger() *zap.Logger {
	var zapLogger *zap.Logger
	var err error

	env := os.Getenv("ENV")
	if env == "production" {
		zapLogger, err = zap.NewProduction()
	} else {
		zapLogger, err = zap.NewDevelopment()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Créer le handler slog qui délègue vers zap
	handler := NewZapHandler(zapLogger)

	// Initialiser le logger slog global
	SlogLog = slog.New(handler)
	slog.SetDefault(SlogLog)

	return zapLogger
}

// GetSlog retourne l'instance globale du logger slog.
func GetSlog() *slog.Logger {
	if SlogLog == nil {
		// Fallback si GetSlog est appelé avant InitLogger
		return slog.Default()
	}
	return SlogLog
}
