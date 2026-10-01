// Package logger initialise zap et le logger slog qui lui délègue.
package logger

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/zap"
)

// InitLogger construit le logger zap (production si ENV=production) et le
// logger slog qui lui délègue, puis fait de ce dernier le logger par défaut de
// slog. Les deux loggers sont retournés pour être injectés.
func InitLogger() (*zap.Logger, *slog.Logger, error) {
	config := zap.NewDevelopmentConfig()
	if os.Getenv("ENV") == "production" {
		config = zap.NewProductionConfig()
	}
	// Pile d'appels seulement à partir d'Error : en développement zap en ajoute
	// par défaut dès Warn, donc à chaque 4xx du journal d'accès.
	zapLogger, err := config.Build(zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		return nil, nil, fmt.Errorf("build zap logger: %w", err)
	}

	slogLogger := slog.New(NewZapHandler(zapLogger))
	slog.SetDefault(slogLogger)
	return zapLogger, slogLogger, nil
}
