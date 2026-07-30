package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger логирует каждый запрос структурированной строкой (метод,
// путь, статус, задержка, ID пользователя при наличии) — до этого middleware
// не было вовсе, и единственная видимость запросов давали метрики Prometheus.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency_ms", time.Since(start).Milliseconds(),
		}
		if userID, ok := UserID(c); ok {
			attrs = append(attrs, "user_id", userID)
		}
		slog.Info("request", attrs...)

		return err
	}
}
