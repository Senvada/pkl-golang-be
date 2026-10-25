package auth

import (
	"strings"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/response"
	"pkl-golang-be/internal/shared/validator"

	"github.com/gofiber/fiber/v2"
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

func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data permintaan tidak valid", err)
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi form pendaftaran gagal", errs)
	}

	isProd := h.cfg.AppEnv == "production"
	authResp, refreshToken, err := h.service.Register(c.Context(), req, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return err
	}

	SetRefreshTokenCookie(c, refreshToken, false, isProd)
	return response.Created(c, authResp, "Pendaftaran akun berhasil")
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data permintaan tidak valid", err)
	}

	req.Email = strings.TrimSpace(req.Email)

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Validasi form masuk gagal", errs)
	}

	isProd := h.cfg.AppEnv == "production"
	authResp, refreshToken, err := h.service.Login(c.Context(), req, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return err
	}

	SetRefreshTokenCookie(c, refreshToken, req.Remember, isProd)
	return response.Single(c, authResp, "Masuk berhasil")
}

func (h *Handler) GoogleLogin(c *fiber.Ctx) error {
	var req GoogleAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data permintaan tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Kredensial token Google diperlukan", errs)
	}

	isProd := h.cfg.AppEnv == "production"
	authResp, refreshToken, err := h.service.GoogleLogin(c.Context(), req.Credential, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return err
	}

	SetRefreshTokenCookie(c, refreshToken, false, isProd)
	return response.Single(c, authResp, "Masuk dengan Google berhasil")
}

func (h *Handler) GoogleBind(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Pengguna belum diautentikasi")
	}

	var req GoogleAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.BadRequest("Format data permintaan tidak valid", err)
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return apperror.Validation("Kredensial token Google diperlukan", errs)
	}

	if err := h.service.GoogleBind(c.Context(), claims.UserID, req.Credential); err != nil {
		return err
	}

	return response.Single(c, nil, "Akun Google berhasil ditautkan ke profil Anda")
}

func (h *Handler) GoogleUnbind(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Pengguna belum diautentikasi")
	}

	if err := h.service.GoogleUnbind(c.Context(), claims.UserID); err != nil {
		return err
	}

	return response.Empty(c, "Tautan akun Google berhasil dilepas")
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	refreshToken := GetRefreshToken(c)
	isProd := h.cfg.AppEnv == "production"

	tokenResp, newRefreshToken, err := h.service.Refresh(c.Context(), refreshToken, c.IP(), c.Get("User-Agent"))
	if err != nil {
		ClearRefreshTokenCookie(c, isProd)
		return err
	}

	SetRefreshTokenCookie(c, newRefreshToken, false, isProd)
	return response.Single(c, tokenResp, "Sesi berhasil diperbarui")
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	refreshToken := GetRefreshToken(c)
	isProd := h.cfg.AppEnv == "production"

	_ = h.service.Logout(c.Context(), refreshToken)
	ClearRefreshTokenCookie(c, isProd)

	return response.Empty(c, "Logout berhasil")
}

func (h *Handler) Me(c *fiber.Ctx) error {
	claims, ok := middleware.GetAuthUser(c)
	if !ok {
		return apperror.Unauthorized("Pengguna belum diautentikasi")
	}

	profile, err := h.service.GetMe(c.Context(), claims.UserID)
	if err != nil {
		return err
	}

	return response.Single(c, profile, "Profil pengguna berhasil diambil")
}
