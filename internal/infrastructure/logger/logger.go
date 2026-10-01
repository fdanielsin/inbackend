// Package logger initialise zap et le logger slog qui lui délègue.
package logger

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/zap"
)

// SlogLog est le logger slog global, initialisé par InitLogger.
var SlogLog *slog.Logger

// InitLogger initialise les deux loggers : zap et slog.
// slog délègue vers zap en arrière-plan.
func InitLogger() *zap.Logger {
	var zapLogger *zap.Logger
	var err error

	config := zap.NewDevelopmentConfig()
	if os.Getenv("ENV") == "production" {
		config = zap.NewProductionConfig()
	}
	// Pile d'appels seulement à partir d'Error : en développement zap en ajoute
	// par défaut dès Warn, donc à chaque 4xx du journal d'accès.
	zapLogger, err = config.Build(zap.AddStacktrace(zap.ErrorLevel))
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
