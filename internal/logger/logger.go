package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func InitLogger() *zap.Logger {
	var logger *zap.Logger
	var err error

	env := os.Getenv("ENV")
	if env == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	return logger
}
