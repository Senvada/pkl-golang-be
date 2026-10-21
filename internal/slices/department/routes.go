package department

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Plural routes (/departments)
	departmentsGroup := router.Group("/departments")
	// Public routes
	departmentsGroup.Get("", h.GetAll)
	departmentsGroup.Get("/:id", h.GetByID)

	// Protected routes
	departmentsGroup.Post("", authMiddleware, h.Create)
	departmentsGroup.Put("/:id", authMiddleware, h.Update)
	departmentsGroup.Delete("/:id", authMiddleware, h.Delete)
}
