package user

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

// -----------------------------------------------------------------------
// SELF PROFILE ENDPOINTS
// -----------------------------------------------------------------------

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

// -----------------------------------------------------------------------
// MANAGEMENT ENDPOINTS
// -----------------------------------------------------------------------

func (h *Handler) GetUsers(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var filter UserFilterQuery
	filter.Search = c.Query("search")
	filter.Status = c.Query("status")

	if roleIDStr := c.Query("role_id"); roleIDStr != "" {
		if rID, err := uuid.Parse(roleIDStr); err == nil {
			filter.RoleID = &rID
		}
	}
	if deptIDStr := c.Query("department_id"); deptIDStr != "" {
		if dID, err := uuid.Parse(deptIDStr); err == nil {
			filter.DepartmentID = &dID
		}
	}
	if compIDStr := c.Query("company_id"); compIDStr != "" {
		if cID, err := uuid.Parse(compIDStr); err == nil {
			filter.CompanyID = &cID
		}
	}
	if parentIDStr := c.Query("parent_id"); parentIDStr != "" {
		if pID, err := uuid.Parse(parentIDStr); err == nil {
			filter.ParentID = &pID
		}
	}

	p := pagination.FromContext(c)
	users, meta, err := h.service.GetUsers(c.Context(), claims, filter, p)
	if err != nil {
		return err
	}

	return response.Paginated(c, users, meta, "Daftar pengguna berhasil diambil")
}

func (h *Handler) GetUserByID(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID pengguna tidak valid", err)
	}

	user, err := h.service.GetUserByID(c.Context(), claims, id)
	if err != nil {
		return err
	}

	return response.Single(c, user, "Detail pengguna berhasil diambil")
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pendaftaran pengguna gagal", errs)
	}

	user, err := h.service.CreateUser(c.Context(), claims, req)
	if err != nil {
		return err
	}

	return response.Created(c, user, "Pengguna baru berhasil ditambahkan")
}

func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID pengguna tidak valid", err)
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format request tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi pembaruan pengguna gagal", errs)
	}

	user, err := h.service.UpdateUser(c.Context(), claims, id, req)
	if err != nil {
		return err
	}

	return response.Single(c, user, "Data pengguna berhasil diperbarui")
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok || claims == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.BadRequest("ID pengguna tidak valid", err)
	}

	if err := h.service.DeleteUser(c.Context(), claims, id); err != nil {
		return err
	}

	return response.Single(c, fiber.Map{"deleted": true}, "Pengguna berhasil dihapus")
}
