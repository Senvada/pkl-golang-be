package article

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)
	optionalAuth := middleware.NewOptionalAuthMiddleware(cfg)

	articlesGroup := router.Group("/articles")

	// Public reading routes (with optional auth to detect logged in user role & scope)
	articlesGroup.Get("", optionalAuth, h.GetArticles)
	articlesGroup.Get("/slug/:slug", optionalAuth, h.GetBySlug)
	articlesGroup.Get("/:id", optionalAuth, h.GetByID)

	// Protected management routes
	articlesGroup.Post("", authMiddleware, h.Create)
	articlesGroup.Post("/cover/presign", authMiddleware, h.GenerateCoverPresignedURL)
	articlesGroup.Post("/:id/cover/presign", authMiddleware, h.GenerateCoverPresignedURL)
	articlesGroup.Put("/:id", authMiddleware, h.Update)
	articlesGroup.Patch("/:id/status", authMiddleware, h.UpdateStatus)
	articlesGroup.Delete("/:id", authMiddleware, h.Delete)
}
