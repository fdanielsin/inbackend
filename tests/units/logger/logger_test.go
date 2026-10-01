// Package logger tests the zap initialisation and the slog handler that delegates to zap.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func observed() (*slog.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return slog.New(logger.NewZapHandler(zap.New(core))), logs
}

// GIVEN slog levels, standard or in between / WHEN logged / THEN each is written at the zap level of its range, none is lost
func TestZapHandler_MapsEveryLevel(t *testing.T) {
	log, logs := observed()
	cases := []struct {
		level slog.Level
		want  zapcore.Level
	}{
		{slog.Level(-8), zapcore.DebugLevel},
		{slog.LevelDebug, zapcore.DebugLevel},
		{slog.Level(-1), zapcore.DebugLevel},
		{slog.LevelInfo, zapcore.InfoLevel},
		{slog.Level(2), zapcore.InfoLevel},
		{slog.LevelWarn, zapcore.WarnLevel},
		{slog.Level(6), zapcore.WarnLevel},
		{slog.LevelError, zapcore.ErrorLevel},
		{slog.Level(12), zapcore.ErrorLevel},
	}

	for _, c := range cases {
		log.Log(context.Background(), c.level, "entry")
	}

	entries := logs.All()
	require.Len(t, entries, len(cases), "no entry may be dropped")
	for i, c := range cases {
		assert.Equal(t, c.want, entries[i].Level, "slog level %d", c.level)
	}
}

// GIVEN attributes of every kind / WHEN logged / THEN they reach zap with their type
func TestZapHandler_KeepsTypedAttributes(t *testing.T) {
	log, logs := observed()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	log.Info("typed",
		slog.String("s", "text"),
		slog.Int64("i", -3),
		slog.Uint64("u", 7),
		slog.Float64("f", 1.5),
		slog.Bool("b", true),
		slog.Duration("d", 2*time.Second),
		slog.Time("t", at),
		slog.Any("a", []int{1, 2}),
	)

	fields := logs.All()[0].ContextMap()
	assert.Equal(t, "text", fields["s"])
	assert.Equal(t, int64(-3), fields["i"])
	assert.Equal(t, uint64(7), fields["u"])
	assert.InDelta(t, 1.5, fields["f"], 0)
	assert.Equal(t, true, fields["b"])
	assert.Equal(t, 2*time.Second, fields["d"])
	assert.Equal(t, at, fields["t"])
	assert.Equal(t, []any{1, 2}, fields["a"], "zap.Any keeps the values")
}

// GIVEN a logger with attributes and a group / WHEN logged / THEN the attributes are kept and the group namespaces the rest
func TestZapHandler_WithAttrsAndGroup(t *testing.T) {
	log, logs := observed()

	log.With(slog.String("request_id", "r-1")).WithGroup("http").Info("grouped", slog.Int("status", 200))

	fields := logs.All()[0].ContextMap()
	assert.Equal(t, "r-1", fields["request_id"])
	assert.Equal(t, map[string]any{"status": int64(200)}, fields["http"])
}

// GIVEN any level / THEN the handler reports it enabled and lets zap filter
func TestZapHandler_EnabledForEveryLevel(t *testing.T) {
	handler := logger.NewZapHandler(zap.NewNop())

	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		assert.True(t, handler.Enabled(context.Background(), level))
	}
}

// captureStderr runs fn while os.Stderr is a pipe, and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = original }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

// GIVEN ENV=development or production / WHEN InitLogger / THEN slog is set up to write through zap,
// with a stack trace on errors only (not on the warnings of every 4xx)
func TestInitLogger_WritesThroughZapWithStackTracesOnErrorsOnly(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })

	for _, env := range []string{"development", "production"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)

			warnOut := captureStderr(t, func() {
				zapLogger := logger.InitLogger()
				require.NotNil(t, zapLogger)
				assert.Same(t, logger.SlogLog, logger.GetSlog())
				assert.Same(t, logger.SlogLog, slog.Default())
				slog.Warn("warn-entry")
				_ = zapLogger.Sync()
			})
			errorOut := captureStderr(t, func() {
				zapLogger := logger.InitLogger()
				slog.Error("error-entry", slog.String("error_code", "INTERNAL_ERROR"))
				_ = zapLogger.Sync()
			})

			assert.Equal(t, env == "production", strings.HasPrefix(warnOut, "{"), "JSON lines in production only:\n%s", warnOut)
			assert.Contains(t, warnOut, "warn-entry")
			assert.NotContains(t, warnOut, "logger_test.go", "no stack trace on a warning")
			assert.Contains(t, errorOut, "error-entry")
			assert.Contains(t, errorOut, "INTERNAL_ERROR")
			assert.Contains(t, errorOut, "logger_test.go", "an error carries its stack trace")
		})
	}
}

// GIVEN InitLogger has not run / WHEN GetSlog / THEN slog.Default is returned
func TestGetSlog_BeforeInitFallsBackToDefault(t *testing.T) {
	saved := logger.SlogLog
	logger.SlogLog = nil
	t.Cleanup(func() { logger.SlogLog = saved })

	assert.Same(t, slog.Default(), logger.GetSlog())
}
