// Package middleware содержит сквозные middleware Fiber: JWT-аутентификацию,
// ограничение частоты запросов и обработку ошибок. Бизнес-логики здесь нет.
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/meindokuse/task-service/internal/pkg/jwtutil"
)

const userIDLocalsKey = "user_id"

// Auth проверяет Bearer JWT-токен и сохраняет ID вызывающего пользователя в
// locals под ключом userIDLocalsKey, откуда его можно получить через UserID.
func Auth(issuer *jwtutil.Issuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}

		token := strings.TrimPrefix(header, prefix)
		claims, err := issuer.Parse(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(userIDLocalsKey, claims.UserID)
		return c.Next()
	}
}

// UserID возвращает ID аутентифицированного пользователя, установленный в Auth.
func UserID(c *fiber.Ctx) (uint64, bool) {
	id, ok := c.Locals(userIDLocalsKey).(uint64)
	return id, ok
}
