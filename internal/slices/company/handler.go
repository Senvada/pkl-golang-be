package company

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

func (h *Handler) GetCompanies(c *fiber.Ctx) error {
	var filter CompanyFilterQuery
	filter.Search = c.Query("search")
	filter.Status = c.Query("status")

	if deptIDStr := c.Query("department_id"); deptIDStr != "" {
		if dID, err := uuid.Parse(deptIDStr); err == nil {
			filter.DepartmentID = &dID
		}
	}

	p := pagination.FromContext(c)
	companies, meta, err := h.service.GetCompanies(c.Context(), filter, p)
	if err != nil {
		return err
	}

	return response.Paginated(c, companies, meta, "Daftar perusahaan mitra berhasil diambil")
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

func (h *Handler) GetMaps(c *fiber.Ctx) error {
	mapsData, err := h.service.GetMaps(c.Context())
	if err != nil {
		return err
	}

	return response.Single(c, mapsData, "Data peta mitra industri berhasil diambil")
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

	company, err := h.service.Create(c.Context(), claims, req)
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

	company, err := h.service.Update(c.Context(), claims, id, req)
	if err != nil {
		return err
	}

	return response.Single(c, company, "Perusahaan mitra berhasil diperbarui")
}

func (h *Handler) UpdateContent(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID perusahaan tidak valid", err)
	}

	var req UpdateCompanyContentRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi konten gagal", errs)
	}

	company, err := h.service.UpdateContent(c.Context(), claims, id, req.Content)
	if err != nil {
		return err
	}

	return response.Single(c, company, "Konten profil perusahaan berhasil diperbarui")
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

	if err := h.service.Delete(c.Context(), claims, id); err != nil {
		return err
	}

	return response.Single(c, fiber.Map{"deleted": true}, "Perusahaan mitra berhasil dihapus")
}

func (h *Handler) PresignImage(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var id uuid.UUID
	idStr := c.Params("id")
	if idStr != "" {
		var err error
		id, err = uuid.Parse(idStr)
		if err != nil {
			return apperror.BadRequest("ID perusahaan tidak valid", err)
		}
	}

	var req PresignImageRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi permintaan upload gambar gagal", errs)
	}

	presignResp, err := h.service.GenerateImagePresignedURL(c.Context(), claims, id, req)
	if err != nil {
		return err
	}

	return response.Single(c, presignResp, "Presigned upload URL foto perusahaan berhasil diterbitkan")
}

func (h *Handler) ResolveMaps(c *fiber.Ctx) error {
	var req ResolveMapsRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi URL Google Maps gagal", errs)
	}

	coords, err := h.service.ResolveMaps(c.Context(), req.URL)
	if err != nil {
		return err
	}

	return response.Single(c, coords, "Koordinat lokasi berhasil didapatkan")
}
