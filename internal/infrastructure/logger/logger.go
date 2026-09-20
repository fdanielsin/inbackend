package logger

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/zap"
)

// Global logger instances
var (
	ZapLog  *zap.Logger
	SlogLog *slog.Logger
)

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

	// Stocker globalement
	ZapLog = zapLogger

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

// GetZap retourne l'instance globale du logger zap.
func GetZap() *zap.Logger {
	if ZapLog == nil {
		// Fallback si GetZap est appelé avant InitLogger
		ZapLog, _ = zap.NewDevelopment()
	}
	return ZapLog
}
