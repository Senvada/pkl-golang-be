package company

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Plural routes (/companies)
	companiesGroup := router.Group("/companies", authMiddleware)
	companiesGroup.Get("", h.GetAll)
	companiesGroup.Get("/:id", h.GetByID)
	companiesGroup.Get("/slug/:slug", h.GetBySlug)
	companiesGroup.Post("", h.Create)
	companiesGroup.Put("/:id", h.Update)
	companiesGroup.Delete("/:id", h.Delete)

	// Singular routes compatibility (/company)
	companyGroup := router.Group("/company", authMiddleware)
	companyGroup.Get("", h.GetAll)
	companyGroup.Get("/:id", h.GetByID)
	companyGroup.Get("/slug/:slug", h.GetBySlug)
	companyGroup.Post("", h.Create)
	companyGroup.Put("/:id", h.Update)
	companyGroup.Delete("/:id", h.Delete)
}
