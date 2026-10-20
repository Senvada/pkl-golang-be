package department

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Plural routes (/departments)
	departmentsGroup := router.Group("/departments", authMiddleware)
	departmentsGroup.Get("", h.GetAll)
	departmentsGroup.Get("/:id", h.GetByID)
	departmentsGroup.Post("", h.Create)
	departmentsGroup.Put("/:id", h.Update)
	departmentsGroup.Delete("/:id", h.Delete)
}
