package logger

import (
	"context"
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapHandler implémente slog.Handler en déléguant vers zap.
type ZapHandler struct {
	log *zap.Logger
}

// NewZapHandler crée un handler slog qui délègue vers zap.
func NewZapHandler(logger *zap.Logger) *ZapHandler {
	return &ZapHandler{log: logger}
}

// Handle traite un enregistrement slog et le délègue vers zap.
func (h *ZapHandler) Handle(ctx context.Context, r slog.Record) error {
	// Construire les champs zap depuis les attributs slog
	fields := make([]zap.Field, 0, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		fields = append(fields, zapFieldFromAttr(a))
		return true
	})

	// Déterminer le niveau et logger
	switch r.Level {
	case slog.LevelDebug:
		h.log.Debug(r.Message, fields...)
	case slog.LevelInfo:
		h.log.Info(r.Message, fields...)
	case slog.LevelWarn:
		h.log.Warn(r.Message, fields...)
	case slog.LevelError:
		h.log.Error(r.Message, fields...)
	}

	return nil
}

// WithAttrs retourne un nouveau handler avec des attributs supplémentaires.
func (h *ZapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	fields := make([]zap.Field, 0, len(attrs))
	for _, a := range attrs {
		fields = append(fields, zapFieldFromAttr(a))
	}
	return &ZapHandler{log: h.log.With(fields...)}
}

// WithGroup retourne un nouveau handler avec un groupe (namespace) d'attributs.
func (h *ZapHandler) WithGroup(name string) slog.Handler {
	return &ZapHandler{log: h.log.With(zap.Namespace(name))}
}

// Enabled retourne vrai si le niveau de log doit être traité.
func (h *ZapHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true // zap filtre déjà par niveau
}

// zapFieldFromAttr convertit un slog.Attr en zap.Field.
func zapFieldFromAttr(a slog.Attr) zap.Field {
	switch a.Value.Kind() {
	case slog.KindString:
		return zap.String(a.Key, a.Value.String())
	case slog.KindInt64:
		return zap.Int64(a.Key, a.Value.Int64())
	case slog.KindUint64:
		return zap.Uint64(a.Key, a.Value.Uint64())
	case slog.KindFloat64:
		return zap.Float64(a.Key, a.Value.Float64())
	case slog.KindBool:
		return zap.Bool(a.Key, a.Value.Bool())
	case slog.KindDuration:
		return zap.Duration(a.Key, a.Value.Duration())
	case slog.KindTime:
		return zap.Time(a.Key, a.Value.Time())
	default:
		return zap.Any(a.Key, a.Value.Any())
	}
}

// ZapLevelForSlog retourne le niveau zap correspondant au niveau slog.
func zapLevelForSlog(level slog.Level) zapcore.Level {
	switch level {
	case slog.LevelDebug:
		return zapcore.DebugLevel
	case slog.LevelInfo:
		return zapcore.InfoLevel
	case slog.LevelWarn:
		return zapcore.WarnLevel
	case slog.LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
