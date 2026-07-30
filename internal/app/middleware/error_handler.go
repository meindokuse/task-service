package middleware

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type errorResponse struct {
	Error string `json:"error"`
}

// ErrorHandler преобразует доменные ошибки (terror) и ошибки Fiber в JSON-тело
// ответа с корректным HTTP-статусом, логируя ответы 5xx как Error, а 4xx — как
// Warn (ранее логировался только код >= 500, а ветка *fiber.Error не
// логировалась вовсе).
func ErrorHandler(c *fiber.Ctx, err error) error {
	var status int
	var message string

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		status = fiberErr.Code
		message = fiberErr.Message
	} else {
		status = terror.HTTPStatus(err)
		message = err.Error()
	}

	switch {
	case status >= 500:
		slog.Error("internal error", "error", err, "path", c.Path(), "method", c.Method())
	case status >= 400:
		slog.Warn("request error", "error", err, "path", c.Path(), "method", c.Method(), "status", status)
	}

	return c.Status(status).JSON(errorResponse{Error: message})
}
