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
// ответа с корректным HTTP-статусом, логируя при этом ответы с кодом 5xx.
func ErrorHandler(c *fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(errorResponse{Error: fiberErr.Message})
	}

	status := terror.HTTPStatus(err)
	if status >= 500 {
		slog.Error("internal error", "error", err, "path", c.Path())
	}

	return c.Status(status).JSON(errorResponse{Error: err.Error()})
}
