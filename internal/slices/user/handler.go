package user

import (
	"strings"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/response"
	"pkl-golang-be/internal/shared/validator"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
	cfg     *config.Config
}

func NewHandler(service Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	profile, err := h.service.GetProfile(c.Context(), claims.UserID)
	if err != nil {
		return err
	}

	return response.Single(c, profile, "Profil pengguna berhasil diambil")
}

func (h *Handler) UpdateMe(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan profil gagal", errs)
	}

	profile, err := h.service.UpdateProfile(c.Context(), claims.UserID, req)
	if err != nil {
		return err
	}

	return response.Single(c, profile, "Profil pengguna berhasil diperbarui")
}

func (h *Handler) UpdateByID(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	targetIDStr := c.Params("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return apperror.BadRequest("ID pengguna tidak valid", err)
	}

	// Active user can update themselves; admins can update any user
	if claims.UserID != targetID && !strings.EqualFold(claims.RoleName, "admin") {
		return apperror.Forbidden("Anda tidak memiliki izin untuk mengubah data pengguna ini")
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan profil gagal", errs)
	}

	profile, err := h.service.UpdateProfile(c.Context(), targetID, req)
	if err != nil {
		return err
	}

	return response.Single(c, profile, "Profil pengguna berhasil diperbarui")
}

func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi ubah kata sandi gagal", errs)
	}

	if err := h.service.ChangePassword(c.Context(), claims.UserID, req); err != nil {
		return err
	}

	return response.Single(c, fiber.Map{"updated": true}, "Kata sandi berhasil diperbarui")
}

func (h *Handler) PresignAvatar(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req PresignAvatarRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembuatan presigned URL gagal", errs)
	}

	presignResp, err := h.service.GenerateAvatarPresignedURL(c.Context(), claims.UserID, req)
	if err != nil {
		return err
	}

	return response.Single(c, presignResp, "Presigned upload URL berhasil diterbitkan")
}

func (h *Handler) DeleteAvatar(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	profile, err := h.service.DeleteAvatar(c.Context(), claims.UserID)
	if err != nil {
		return err
	}

	return response.Single(c, profile, "Foto profil berhasil dihapus")
}
