// Package logger централизованно настраивает slog: без него cfg.LogLevel
// нигде не применялся, а вызовы slog.Info/Warn/Error по всему проекту
// работали на дефолтном текстовом хендлере с уровнем Info.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// ParseLevel переводит строковый уровень логирования (регистронезависимо)
// в slog.Level, по умолчанию (в том числе для нераспознанного значения) — Info.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Init настраивает slog.Default на JSON-хендлер с уровнем, разобранным из
// level (значение cfg.LogLevel), и должен вызываться как можно раньше в
// цепочке инициализации, чтобы все последующие вызовы slog.* его учитывали.
func Init(level string) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: ParseLevel(level)})
	slog.SetDefault(slog.New(handler))
}
