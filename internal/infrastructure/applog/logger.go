// Package applog construit le logger slog de l'application.
package applog

import (
	"io"
	"log/slog"
	"os"
)

const productionEnv = "production"

// New construit le logger : JSON au niveau Info quand env vaut "production",
// texte au niveau Debug sinon. Chaque ligne porte sa position dans le code source.
func New(w io.Writer, env string) *slog.Logger {
	if env == productionEnv {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}))
}

// Init construit le logger de l'application sur stderr, selon ENV, et en
// fait le logger par défaut de slog.
func Init() *slog.Logger {
	log := New(os.Stderr, os.Getenv("ENV"))
	slog.SetDefault(log)
	return log
}
