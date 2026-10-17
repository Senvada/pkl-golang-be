package asset

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/pagination"
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

func (h *Handler) PresignUpload(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req PresignAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi presign upload gagal", errs)
	}

	res, err := h.service.PresignUpload(c.Context(), claims, req)
	if err != nil {
		return err
	}

	return response.Single(c, res, "Presigned upload URL berhasil dibuat")
}

func (h *Handler) Create(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req CreateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi data aset gagal", errs)
	}

	res, err := h.service.CreateAsset(c.Context(), claims, req)
	if err != nil {
		return err
	}

	return response.Created(c, res, "Aset berhasil disimpan")
}

func (h *Handler) GetAll(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	p := pagination.FromContext(c)
	filter := AssetFilterQuery{
		MimeType: c.Query("mime_type"),
		Search:   p.Search,
	}

	items, meta, err := h.service.GetAssets(c.Context(), claims, filter, p)
	if err != nil {
		return err
	}

	return response.Paginated(c, items, meta, "Daftar aset berhasil diambil")
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperror.BadRequest("Format ID aset tidak valid", err)
	}

	res, err := h.service.GetAssetByID(c.Context(), claims, id)
	if err != nil {
		return err
	}

	return response.Single(c, res, "Data aset berhasil diambil")
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperror.BadRequest("Format ID aset tidak valid", err)
	}

	if err := h.service.DeleteAsset(c.Context(), claims, id); err != nil {
		return err
	}

	return response.Empty(c, "Aset berhasil dihapus")
}
