package role

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Plural routes (/roles/...)
	rolesGroup := router.Group("/roles", authMiddleware)
	rolesGroup.Get("/", h.GetAll)
	rolesGroup.Post("/", h.Create)
	rolesGroup.Get("/:id", h.GetByID)
	rolesGroup.Put("/:id", h.Update)
	rolesGroup.Patch("/:id", h.Update)
	rolesGroup.Delete("/:id", h.Delete)
}
