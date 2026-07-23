package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/meindokuse/task-service/config"
)

func limitReached(c *fiber.Ctx) error {
	return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded, try again later")
}

// RateLimitByUser применяет лимит запросов на пользователя; должен выполняться
// после Auth, чтобы ID пользователя уже был установлен в locals.
func RateLimitByUser() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.RateLimitRequestsPerMinute,
		Expiration: config.RateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			if id, ok := UserID(c); ok {
				return "user:" + strconv.FormatUint(id, 10)
			}
			return "ip:" + c.IP()
		},
		LimitReached: limitReached,
	})
}

// RateLimitByIP защищает неаутентифицированные эндпоинты (register/login) тем же
// лимитом, но по IP клиента, так как ID пользователя ещё не известен.
func RateLimitByIP() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.RateLimitRequestsPerMinute,
		Expiration: config.RateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ip:" + c.IP()
		},
		LimitReached: limitReached,
	})
}
