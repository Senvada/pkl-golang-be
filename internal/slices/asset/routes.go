package asset

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Plural routes (/assets)
	assetsGroup := router.Group("/assets", authMiddleware)
	assetsGroup.Get("", h.GetAll)
	assetsGroup.Get("/:id", h.GetByID)
	assetsGroup.Post("/presign", h.PresignUpload)
	assetsGroup.Post("", h.Create)
	assetsGroup.Delete("/:id", h.Delete)
}
