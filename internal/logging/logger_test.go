package logging_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/audetv/mailbridge/internal/logging"
)

func TestNew_JSON(t *testing.T) {
	ctx := context.TODO()

	logger := logging.New("info", "json")
	if logger == nil {
		t.Fatal("logger is nil")
	}

	if !logger.Enabled(ctx, slog.LevelInfo) {
		t.Error("info level should be enabled")
	}
	if logger.Enabled(ctx, slog.LevelDebug) {
		t.Error("debug level should be disabled for info")
	}
}

func TestNew_Text(t *testing.T) {
	ctx := context.TODO()

	logger := logging.New("debug", "text")
	if logger == nil {
		t.Fatal("logger is nil")
	}

	if !logger.Enabled(ctx, slog.LevelDebug) {
		t.Error("debug level should be enabled")
	}
}

func TestParseLevel(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		logger := logging.New(tt.input, "text")
		if !logger.Enabled(ctx, tt.expected) {
			t.Errorf("level %s should be enabled for input %s", tt.expected, tt.input)
		}
	}
}

func TestNew_DisabledDebug(t *testing.T) {
	ctx := context.TODO()

	logger := logging.New("warn", "json")

	if logger.Enabled(ctx, slog.LevelDebug) {
		t.Error("debug should be disabled when level is warn")
	}
	if logger.Enabled(ctx, slog.LevelInfo) {
		t.Error("info should be disabled when level is warn")
	}
	if !logger.Enabled(ctx, slog.LevelWarn) {
		t.Error("warn should be enabled when level is warn")
	}
}
