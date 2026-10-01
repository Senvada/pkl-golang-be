package middleware

import (
	"pkl-golang-be/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func NewCORS(cfg *config.Config) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	})
}
