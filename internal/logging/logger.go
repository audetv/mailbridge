// Package logging предоставляет настройку структурированного логирования.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New создаёт новый slog.Logger на основе конфигурации.
func New(level, format string) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// parseLevel преобразует строку в slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
