package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		latency := time.Since(start).Milliseconds()

		var userID string
		if val := c.Locals("user_id"); val != nil {
			userID, _ = val.(string)
		}

		status := c.Response().StatusCode()

		slog.Info("http request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Int64("latency_ms", latency),
			slog.String("user_id", userID),
			slog.String("ip", c.IP()),
		)

		return err
	}
}
