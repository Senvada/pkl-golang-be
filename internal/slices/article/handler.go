package article

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

func (h *Handler) GetArticles(c *fiber.Ctx) error {
	actor, _ := middleware.GetAuthUser(c)

	var filter ArticleFilterQuery
	filter.Search = c.Query("search")
	filter.Status = c.Query("status")

	if cIDStr := c.Query("company_id"); cIDStr != "" {
		if cID, err := uuid.Parse(cIDStr); err == nil {
			filter.CompanyID = &cID
		}
	}

	if uIDStr := c.Query("user_id"); uIDStr != "" {
		if uID, err := uuid.Parse(uIDStr); err == nil {
			filter.UserID = &uID
		}
	}

	p := pagination.FromContext(c)
	articles, meta, err := h.service.GetArticles(c.Context(), actor, filter, p)
	if err != nil {
		return err
	}

	return response.Paginated(c, articles, meta, "Daftar artikel berhasil diambil")
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID artikel tidak valid", err)
	}

	art, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return err
	}

	return response.Single(c, art, "Data artikel berhasil diambil")
}

func (h *Handler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return apperror.BadRequest("Slug artikel tidak valid", nil)
	}

	art, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return err
	}

	return response.Single(c, art, "Data artikel berhasil diambil")
}

func (h *Handler) Create(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req CreateArticleRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembuatan artikel gagal", errs)
	}

	art, err := h.service.Create(c.Context(), actor, req)
	if err != nil {
		return err
	}

	return response.Created(c, art, "Artikel baru berhasil dibuat")
}

func (h *Handler) Update(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID artikel tidak valid", err)
	}

	var req UpdateArticleRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan artikel gagal", errs)
	}

	art, err := h.service.Update(c.Context(), actor, id, req)
	if err != nil {
		return err
	}

	return response.Single(c, art, "Artikel berhasil diperbarui")
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID artikel tidak valid", err)
	}

	var req UpdateArticleStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi status artikel gagal", errs)
	}

	art, err := h.service.UpdateStatus(c.Context(), actor, id, req)
	if err != nil {
		return err
	}

	return response.Single(c, art, "Status artikel berhasil diperbarui")
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID artikel tidak valid", err)
	}

	if err := h.service.Delete(c.Context(), actor, id); err != nil {
		return err
	}

	return response.Single(c, nil, "Artikel berhasil dihapus")
}

func (h *Handler) GenerateCoverPresignedURL(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var targetID uuid.UUID
	idStr := c.Params("id")
	if idStr != "" {
		if parsed, err := uuid.Parse(idStr); err == nil {
			targetID = parsed
		}
	}

	var req PresignCoverRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi presigned cover gagal", errs)
	}

	res, err := h.service.GenerateCoverPresignedURL(c.Context(), actor, targetID, req)
	if err != nil {
		return err
	}

	return response.Single(c, res, "Presigned URL cover artikel berhasil dibuat")
}
