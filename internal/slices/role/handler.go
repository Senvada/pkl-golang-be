package role

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

func (h *Handler) checkAuthAndPermission(c *fiber.Ctx, action string) (*middleware.AuthClaims, error) {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return nil, apperror.Unauthorized("Autentikasi diperlukan")
	}

	// Admin has universal permission
	if strings.EqualFold(claims.RoleName, "Admin") {
		return claims, nil
	}

	hasPerm, err := h.service.CheckPermission(c.Context(), claims.RoleID, "role", action)
	if err != nil {
		return nil, apperror.Internal("Gagal memvalidasi hak akses role", err)
	}

	if !hasPerm {
		return nil, apperror.Forbidden("Anda tidak memiliki izin untuk melakukan aksi ini pada role")
	}

	return claims, nil
}

func (h *Handler) GetAll(c *fiber.Ctx) error {
	if _, err := h.checkAuthAndPermission(c, "read"); err != nil {
		return err
	}

	roles, err := h.service.GetAll(c.Context())
	if err != nil {
		return err
	}

	return response.Single(c, roles, "Daftar role berhasil diambil")
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	if _, err := h.checkAuthAndPermission(c, "read"); err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperror.BadRequest("ID role tidak valid", err)
	}

	role, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return err
	}

	return response.Single(c, role, "Detail role berhasil diambil")
}

func (h *Handler) Create(c *fiber.Ctx) error {
	if _, err := h.checkAuthAndPermission(c, "create"); err != nil {
		return err
	}

	var req CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembuatan role gagal", errs)
	}

	role, err := h.service.Create(c.Context(), req)
	if err != nil {
		return err
	}

	return response.Created(c, role, "Role baru berhasil dibuat")
}

func (h *Handler) Update(c *fiber.Ctx) error {
	if _, err := h.checkAuthAndPermission(c, "update"); err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperror.BadRequest("ID role tidak valid", err)
	}

	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan role gagal", errs)
	}

	role, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		return err
	}

	return response.Single(c, role, "Role berhasil diperbarui")
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	if _, err := h.checkAuthAndPermission(c, "delete"); err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperror.BadRequest("ID role tidak valid", err)
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return err
	}

	return response.Empty(c, "Role berhasil dihapus")
}
