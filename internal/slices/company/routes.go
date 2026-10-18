package company

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Direct routes
	router.Post("/maps/resolve", authMiddleware, h.ResolveMaps)

	// Plural routes (/companies)
	companiesGroup := router.Group("/companies")
	// Public routes
	companiesGroup.Get("", h.GetCompanies)
	companiesGroup.Get("/maps", h.GetMaps)
	companiesGroup.Get("/slug/:slug", h.GetBySlug)
	companiesGroup.Get("/:id", h.GetByID)

	// Protected routes
	companiesGroup.Post("", authMiddleware, h.Create)
	companiesGroup.Post("/maps/resolve", authMiddleware, h.ResolveMaps)
	companiesGroup.Post("/image/presign", authMiddleware, h.PresignImage)
	companiesGroup.Post("/:id/image/presign", authMiddleware, h.PresignImage)
	companiesGroup.Put("/:id", authMiddleware, h.Update)
	companiesGroup.Patch("/:id/content", authMiddleware, h.UpdateContent)
	companiesGroup.Delete("/:id", authMiddleware, h.Delete)

	// Singular routes compatibility (/company)
	companyGroup := router.Group("/company")
	companyGroup.Get("", h.GetCompanies)
	companyGroup.Get("/maps", h.GetMaps)
	companyGroup.Get("/slug/:slug", h.GetBySlug)
	companyGroup.Get("/:id", h.GetByID)

	companyGroup.Post("", authMiddleware, h.Create)
	companyGroup.Post("/maps/resolve", authMiddleware, h.ResolveMaps)
	companyGroup.Post("/image/presign", authMiddleware, h.PresignImage)
	companyGroup.Post("/:id/image/presign", authMiddleware, h.PresignImage)
	companyGroup.Put("/:id", authMiddleware, h.Update)
	companyGroup.Patch("/:id/content", authMiddleware, h.UpdateContent)
	companyGroup.Delete("/:id", authMiddleware, h.Delete)

	// Direct compatibility route /maps/resolve
	router.Post("/maps/resolve", authMiddleware, h.ResolveMaps)
}
