package logger

import (
	"context"
	"log/slog"

	"go.uber.org/zap"
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
func (h *ZapHandler) Handle(_ context.Context, r slog.Record) error {
	fields := make([]zap.Field, 0, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		fields = append(fields, zapFieldFromAttr(a))
		return true
	})

	// Chaque niveau slog, y compris intermédiaire (slog.Level(2), slog.Level(12)),
	// est ramené à la tranche zap correspondante : aucune entrée n'est perdue.
	switch {
	case r.Level < slog.LevelInfo:
		h.log.Debug(r.Message, fields...)
	case r.Level < slog.LevelWarn:
		h.log.Info(r.Message, fields...)
	case r.Level < slog.LevelError:
		h.log.Warn(r.Message, fields...)
	default:
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
func (h *ZapHandler) Enabled(_ context.Context, _ slog.Level) bool {
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
