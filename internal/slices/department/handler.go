package department

import (
	"pkl-golang-be/internal/config"
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
	departments, err := h.service.GetAll(c.Context())
	if err != nil {
		return err
	}
	return response.Single(c, departments, "Daftar jurusan berhasil diambil")
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID jurusan tidak valid", err)
	}

	dept, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return err
	}

	return response.Single(c, dept, "Detail jurusan berhasil diambil")
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateDepartmentRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi jurusan gagal", errs)
	}

	dept, err := h.service.Create(c.Context(), req)
	if err != nil {
		return err
	}

	return response.Created(c, dept, "Jurusan berhasil ditambahkan")
}

func (h *Handler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID jurusan tidak valid", err)
	}

	var req UpdateDepartmentRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan jurusan gagal", errs)
	}

	dept, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		return err
	}

	return response.Single(c, dept, "Jurusan berhasil diperbarui")
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID jurusan tidak valid", err)
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return err
	}

	return response.Single(c, fiber.Map{"deleted": true}, "Jurusan berhasil dihapus")
}
