package auth

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authGroup := router.Group("/auth")

	// Public routes
	authGroup.Post("/register", h.Register)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/refresh", h.Refresh)
	authGroup.Post("/logout", h.Logout)
	authGroup.Post("/google/login", h.GoogleLogin)

	// Protected routes
	protected := authGroup.Group("", middleware.NewAuthMiddleware(cfg))
	protected.Get("/me", h.Me)
	protected.Post("/google/bind", h.GoogleBind)
	protected.Delete("/google/unbind", h.GoogleUnbind)
}
