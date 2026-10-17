package user

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Plural users routes (/users/...)
	usersGroup := router.Group("/users", authMiddleware)
	usersGroup.Get("/me", h.GetMe)
	usersGroup.Patch("/me", h.UpdateMe)
	usersGroup.Put("/me/password", h.ChangePassword)
	usersGroup.Post("/me/avatar/presign", h.PresignAvatar)
	usersGroup.Delete("/me/avatar", h.DeleteAvatar)

	usersGroup.Get("", h.GetUsers)
	usersGroup.Post("", h.CreateUser)
	usersGroup.Get("/:id", h.GetUserByID)
	usersGroup.Put("/:id", h.UpdateUser)
	usersGroup.Patch("/:id", h.UpdateUser)
	usersGroup.Delete("/:id", h.DeleteUser)

	// Singular user routes (/user/...)
	userGroup := router.Group("/user", authMiddleware)
	userGroup.Get("/me", h.GetMe)
	userGroup.Patch("/me", h.UpdateMe)
	userGroup.Put("/me/password", h.ChangePassword)
	userGroup.Post("/me/avatar/presign", h.PresignAvatar)
	userGroup.Delete("/me/avatar", h.DeleteAvatar)

	userGroup.Get("", h.GetUsers)
	userGroup.Post("", h.CreateUser)
	userGroup.Get("/:id", h.GetUserByID)
	userGroup.Put("/:id", h.UpdateUser)
	userGroup.Patch("/:id", h.UpdateUser)
	userGroup.Delete("/:id", h.DeleteUser)
}
