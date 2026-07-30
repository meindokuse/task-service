package logger_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meindokuse/task-service/internal/pkg/logger"
)

func TestParseLevel(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{"debug lowercase", "debug", slog.LevelDebug},
		{"debug mixed case", "DeBuG", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"warning alias", "warning", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"empty defaults to info", "", slog.LevelInfo},
		{"unknown defaults to info", "not-a-level", slog.LevelInfo},
		{"padded with whitespace", "  debug  ", slog.LevelDebug},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, logger.ParseLevel(tc.input))
		})
	}
}
