package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
					slog.String("method", c.Method()),
					slog.String("path", c.Path()),
				)
				c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
			}
		}()
		return c.Next()
	}
}
