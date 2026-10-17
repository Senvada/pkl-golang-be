package company

import (
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

func (h *Handler) GetAll(c *fiber.Ctx) error {
	companies, err := h.service.GetAll(c.Context())
	if err != nil {
		return err
	}
	return response.Single(c, companies, "Daftar perusahaan mitra berhasil diambil")
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID perusahaan tidak valid", err)
	}

	company, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return err
	}

	return response.Single(c, company, "Detail perusahaan mitra berhasil diambil")
}

func (h *Handler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return apperror.BadRequest("Slug perusahaan tidak boleh kosong", nil)
	}

	company, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return err
	}

	return response.Single(c, company, "Detail perusahaan mitra berhasil diambil")
}

func (h *Handler) Create(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req CreateCompanyRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi perusahaan mitra gagal", errs)
	}

	company, err := h.service.Create(c.Context(), claims.UserID, req)
	if err != nil {
		return err
	}

	return response.Created(c, company, "Perusahaan mitra berhasil ditambahkan")
}

func (h *Handler) Update(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID perusahaan tidak valid", err)
	}

	var req UpdateCompanyRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan perusahaan mitra gagal", errs)
	}

	company, err := h.service.Update(c.Context(), claims.UserID, id, req)
	if err != nil {
		return err
	}

	return response.Single(c, company, "Perusahaan mitra berhasil diperbarui")
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID perusahaan tidak valid", err)
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return err
	}

	return response.Single(c, fiber.Map{"deleted": true}, "Perusahaan mitra berhasil dihapus")
}
