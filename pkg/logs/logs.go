package logs

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

type multiHandler struct{ hs []slog.Handler }

func (m multiHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, h := range m.hs {
		if h.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (m multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.hs {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r)
		}
	}
	return nil
}

func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, 0, len(m.hs))
	for _, h := range m.hs {
		out = append(out, h.WithAttrs(attrs))
	}
	return multiHandler{hs: out}
}

func (m multiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, 0, len(m.hs))
	for _, h := range m.hs {
		out = append(out, h.WithGroup(name))
	}
	return multiHandler{hs: out}
}

func SetupLogger(levelStr string) {
	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	otelHandler := otelslog.NewHandler(
		"cache-population",
		otelslog.WithSource(true),
	)

	tintHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case "task":
				return tint.Attr(3, a)
			case "action":
				return tint.Attr(2, a)
			case "percentageTransmitted":
				return tint.Attr(5, a)
			}
			return a
		},
	})

	slog.SetDefault(slog.New(multiHandler{hs: []slog.Handler{otelHandler, tintHandler}}))
}
