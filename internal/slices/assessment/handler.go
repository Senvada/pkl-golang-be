package assessment

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

// -----------------------------------------------------------------------------
// Academic Periods Handlers
// -----------------------------------------------------------------------------

func (h *Handler) GetAcademicPeriods(c *fiber.Ctx) error {
	periods, err := h.service.GetAllAcademicPeriods(c.Context())
	if err != nil {
		return err
	}
	return response.Single(c, periods, "Daftar periode akademik berhasil diambil")
}

func (h *Handler) GetActiveAcademicPeriod(c *fiber.Ctx) error {
	period, err := h.service.GetActiveAcademicPeriod(c.Context())
	if err != nil {
		return err
	}
	return response.Single(c, period, "Periode akademik aktif berhasil diambil")
}

func (h *Handler) CreateAcademicPeriod(c *fiber.Ctx) error {
	var req CreateAcademicPeriodRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi periode akademik gagal", errs)
	}

	period, err := h.service.CreateAcademicPeriod(c.Context(), req)
	if err != nil {
		return err
	}
	return response.Created(c, period, "Periode akademik berhasil dibuat")
}

func (h *Handler) SetActiveAcademicPeriod(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID periode akademik tidak valid", err)
	}

	period, err := h.service.SetActiveAcademicPeriod(c.Context(), id)
	if err != nil {
		return err
	}
	return response.Single(c, period, "Periode akademik berhasil diaktifkan")
}

// -----------------------------------------------------------------------------
// Assessment Templates Handlers
// -----------------------------------------------------------------------------

func (h *Handler) GetAssessmentTemplates(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var periodID *uuid.UUID
	if pStr := c.Query("period_id"); pStr != "" {
		if pID, err := uuid.Parse(pStr); err == nil {
			periodID = &pID
		}
	}

	var deptID *uuid.UUID
	if dStr := c.Query("department_id"); dStr != "" {
		if dID, err := uuid.Parse(dStr); err == nil {
			deptID = &dID
		}
	}

	templates, err := h.service.GetAssessmentTemplates(c.Context(), actor.UserID, actor.RoleID, actor.RoleName, periodID, deptID)
	if err != nil {
		return err
	}
	return response.Single(c, templates, "Daftar template penilaian berhasil diambil")
}

func (h *Handler) GetAssessmentTemplateByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID template penilaian tidak valid", err)
	}

	tpl, err := h.service.GetAssessmentTemplateByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.Single(c, tpl, "Template penilaian berhasil diambil")
}

func (h *Handler) CreateAssessmentTemplate(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req CreateAssessmentTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi template penilaian gagal", errs)
	}

	tpl, err := h.service.CreateAssessmentTemplate(c.Context(), actor.UserID, actor.RoleID, actor.RoleName, req)
	if err != nil {
		return err
	}
	return response.Created(c, tpl, "Template penilaian berhasil dibuat")
}

func (h *Handler) UpdateAssessmentTemplate(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID template penilaian tidak valid", err)
	}

	var req UpdateAssessmentTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi template penilaian gagal", errs)
	}

	tpl, err := h.service.UpdateAssessmentTemplate(c.Context(), actor.UserID, actor.RoleID, actor.RoleName, id, req)
	if err != nil {
		return err
	}
	return response.Single(c, tpl, "Template penilaian berhasil diperbarui")
}

// -----------------------------------------------------------------------------
// Student Assessment Handlers
// -----------------------------------------------------------------------------

func (h *Handler) GetMyStudentAssessment(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var periodID *uuid.UUID
	if pStr := c.Query("period_id"); pStr != "" {
		if pID, err := uuid.Parse(pStr); err == nil {
			periodID = &pID
		}
	}

	sa, err := h.service.GetMyStudentAssessment(c.Context(), actor.UserID, periodID)
	if err != nil {
		return err
	}
	return response.Single(c, sa, "Data penilaian siswa berhasil diambil")
}

func (h *Handler) UpsertStudentAssessment(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req UpsertStudentAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi data penilaian gagal", errs)
	}

	sa, err := h.service.UpsertStudentAssessment(c.Context(), actor.UserID, req)
	if err != nil {
		return err
	}
	return response.Single(c, sa, "Penilaian siswa berhasil disimpan")
}

func (h *Handler) GetStudentAssessmentByID(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID penilaian siswa tidak valid", err)
	}

	sa, err := h.service.GetStudentAssessmentByID(c.Context(), id, actor.UserID, actor.RoleID, actor.RoleName)
	if err != nil {
		return err
	}
	return response.Single(c, sa, "Data penilaian berhasil diambil")
}

func (h *Handler) ListStudentAssessments(c *fiber.Ctx) error {
	actor, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var periodID *uuid.UUID
	if pStr := c.Query("period_id"); pStr != "" {
		if pID, err := uuid.Parse(pStr); err == nil {
			periodID = &pID
		}
	}

	var deptID *uuid.UUID
	if dStr := c.Query("department_id"); dStr != "" {
		if dID, err := uuid.Parse(dStr); err == nil {
			deptID = &dID
		}
	}

	search := c.Query("search")

	list, err := h.service.ListStudentAssessments(c.Context(), actor.UserID, actor.RoleID, actor.RoleName, periodID, deptID, search)
	if err != nil {
		return err
	}
	return response.Single(c, list, "Daftar penilaian siswa berhasil diambil")
}
