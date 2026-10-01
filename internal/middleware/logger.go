package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func NewLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(c.Context(), level, "HTTP Request",
			"status", status,
			"method", method,
			"path", path,
			"latency_ms", latency.Milliseconds(),
			"ip", c.IP(),
		)

		return err
	}
}
