package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/shared/apperror"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest, clientIP, userAgent string) (*AuthResponse, string, error)
	Login(ctx context.Context, req LoginRequest, clientIP, userAgent string) (*AuthResponse, string, error)
	GoogleLogin(ctx context.Context, credential string, clientIP, userAgent string) (*AuthResponse, string, error)
	GoogleBind(ctx context.Context, userID uuid.UUID, credential string) error
	GoogleUnbind(ctx context.Context, userID uuid.UUID) error
	Refresh(ctx context.Context, rawRefreshToken string, clientIP, userAgent string) (*RefreshTokenResponse, string, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	GetMe(ctx context.Context, userID uuid.UUID) (*UserProfileDTO, error)
}

type service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) Service {
	return &service{
		db:  db,
		cfg: cfg,
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest, clientIP, userAgent string) (*AuthResponse, string, error) {
	// 1. Verify email uniqueness
	var count int64
	if err := s.db.WithContext(ctx).Model(&User{}).
		Where("email = ? AND deleted_at IS NULL", req.Email).
		Count(&count).Error; err != nil {
		return nil, "", apperror.Internal("Gagal memeriksa ketersediaan email", err)
	}
	if count > 0 {
		return nil, "", apperror.Conflict("Email sudah terdaftar dalam sistem")
	}

	// 2. Verify Role existence
	var role Role
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.RoleID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperror.BadRequest("Peran (role) yang dipilih tidak valid", nil)
		}
		return nil, "", apperror.Internal("Gagal memvalidasi role", err)
	}

	// 3. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, "", apperror.Internal("Gagal memproses kata sandi", err)
	}

	// 4. Create user
	newUser := &User{
		RoleID:       req.RoleID,
		DepartmentID: req.DepartmentID,
		CompanyID:    req.CompanyID,
		Name:         req.Name,
		Email:        req.Email,
		Password:     string(hashedPassword),
		Status:       "active",
	}

	if err := s.db.WithContext(ctx).Create(newUser).Error; err != nil {
		return nil, "", apperror.Internal("Gagal mendaftarkan pengguna baru", err)
	}

	// 5. Generate tokens
	accessToken, expiresIn, err := GenerateAccessToken(newUser, role.Name, s.cfg.JWTSecret)
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan access token", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan refresh token", err)
	}

	// 6. Save refresh token
	tokenRecord := &Token{
		UserID:    newUser.ID,
		TokenHash: HashRefreshToken(refreshToken, s.cfg.JWTSecret),
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
		Remember:  false,
		UserAgent: &userAgent,
		IPAddress: &clientIP,
	}
	if err := s.db.WithContext(ctx).Create(tokenRecord).Error; err != nil {
		return nil, "", apperror.Internal("Gagal menyimpan sesi login", err)
	}

	authResp := &AuthResponse{
		User: UserProfileDTO{
			ID:           newUser.ID,
			Name:         newUser.Name,
			Email:        newUser.Email,
			AvatarURL:    newUser.AvatarURL,
			Status:       newUser.Status,
			RoleID:       newUser.RoleID,
			RoleName:     role.Name,
			DepartmentID: newUser.DepartmentID,
			CompanyID:    newUser.CompanyID,
			GoogleBound:  false,
		},
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	return authResp, refreshToken, nil
}

func (s *service) Login(ctx context.Context, req LoginRequest, clientIP, userAgent string) (*AuthResponse, string, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperror.Unauthorized("Email atau kata sandi tidak cocok")
		}
		return nil, "", apperror.Internal("Gagal memproses login", err)
	}

	if user.Status != "active" {
		return nil, "", apperror.Forbidden(fmt.Sprintf("Akun Anda saat ini berstatus '%s'. Silakan hubungi admin sekolah.", user.Status))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, "", apperror.Unauthorized("Email atau kata sandi tidak cocok")
	}

	var role Role
	_ = s.db.WithContext(ctx).Where("id = ?", user.RoleID).First(&role)

	var googleCount int64
	s.db.WithContext(ctx).Model(&UserGoogleAccount{}).Where("user_id = ?", user.ID).Count(&googleCount)

	accessToken, expiresIn, err := GenerateAccessToken(&user, role.Name, s.cfg.JWTSecret)
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan access token", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan refresh token", err)
	}

	duration := RefreshTokenDuration
	if req.Remember {
		duration = RememberMeDuration
	}

	tokenRecord := &Token{
		UserID:    user.ID,
		TokenHash: HashRefreshToken(refreshToken, s.cfg.JWTSecret),
		ExpiresAt: time.Now().Add(duration),
		Remember:  req.Remember,
		UserAgent: &userAgent,
		IPAddress: &clientIP,
	}
	if err := s.db.WithContext(ctx).Create(tokenRecord).Error; err != nil {
		return nil, "", apperror.Internal("Gagal menyimpan sesi login", err)
	}

	authResp := &AuthResponse{
		User: UserProfileDTO{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			AvatarURL:    user.AvatarURL,
			Status:       user.Status,
			RoleID:       user.RoleID,
			RoleName:     role.Name,
			DepartmentID: user.DepartmentID,
			CompanyID:    user.CompanyID,
			GoogleBound:  googleCount > 0,
		},
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	return authResp, refreshToken, nil
}

func (s *service) GoogleLogin(ctx context.Context, credential string, clientIP, userAgent string) (*AuthResponse, string, error) {
	if s.cfg.GoogleClientID == "" {
		return nil, "", apperror.Internal("Otentikasi Google belum dikonfigurasi di server", nil)
	}

	payload, err := idtoken.Validate(ctx, credential, s.cfg.GoogleClientID)
	if err != nil {
		return nil, "", apperror.Unauthorized("Kredensial token Google tidak valid atau kedaluwarsa")
	}

	sub, _ := payload.Claims["sub"].(string)
	if sub == "" {
		return nil, "", apperror.Unauthorized("Token Google tidak memiliki identifier pengguna (sub)")
	}

	// RULE: Google Auth CANNOT self-register! Must come from a previously bound account.
	var googleAcc UserGoogleAccount
	if err := s.db.WithContext(ctx).Where("google_id = ?", sub).First(&googleAcc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperror.Unauthorized("Akun Google ini belum ditautkan ke akun PKL manapun. Silakan masuk menggunakan email dan password terlebih dahulu, lalu tautkan akun Google Anda di pengaturan profil.")
		}
		return nil, "", apperror.Internal("Gagal memeriksa tautan akun Google", err)
	}

	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", googleAcc.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperror.Unauthorized("Pengguna terkait akun Google ini sudah tidak aktif atau dihapus")
		}
		return nil, "", apperror.Internal("Gagal mengambil data pengguna", err)
	}

	if user.Status != "active" {
		return nil, "", apperror.Forbidden(fmt.Sprintf("Akun Anda saat ini berstatus '%s'. Silakan hubungi admin sekolah.", user.Status))
	}

	var role Role
	_ = s.db.WithContext(ctx).Where("id = ?", user.RoleID).First(&role)

	accessToken, expiresIn, err := GenerateAccessToken(&user, role.Name, s.cfg.JWTSecret)
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan access token", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan refresh token", err)
	}

	tokenRecord := &Token{
		UserID:    user.ID,
		TokenHash: HashRefreshToken(refreshToken, s.cfg.JWTSecret),
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
		Remember:  false,
		UserAgent: &userAgent,
		IPAddress: &clientIP,
	}
	if err := s.db.WithContext(ctx).Create(tokenRecord).Error; err != nil {
		return nil, "", apperror.Internal("Gagal menyimpan sesi login", err)
	}

	authResp := &AuthResponse{
		User: UserProfileDTO{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			AvatarURL:    user.AvatarURL,
			Status:       user.Status,
			RoleID:       user.RoleID,
			RoleName:     role.Name,
			DepartmentID: user.DepartmentID,
			CompanyID:    user.CompanyID,
			GoogleBound:  true,
		},
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	return authResp, refreshToken, nil
}

func (s *service) GoogleBind(ctx context.Context, userID uuid.UUID, credential string) error {
	if s.cfg.GoogleClientID == "" {
		return apperror.Internal("Otentikasi Google belum dikonfigurasi di server", nil)
	}

	payload, err := idtoken.Validate(ctx, credential, s.cfg.GoogleClientID)
	if err != nil {
		return apperror.Unauthorized("Token Google tidak valid")
	}

	sub, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)

	if sub == "" || email == "" {
		return apperror.BadRequest("Data akun Google tidak lengkap", nil)
	}

	// 1. Check if this Google ID is already used by ANOTHER user
	var existingOther UserGoogleAccount
	if err := s.db.WithContext(ctx).Where("google_id = ? AND user_id != ?", sub, userID).First(&existingOther).Error; err == nil {
		return apperror.Conflict("Akun Google ini sudah ditautkan ke akun pengguna lain")
	}

	// 2. Upsert for current user
	var current UserGoogleAccount
	errFind := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&current).Error
	if errFind != nil && !errors.Is(errFind, gorm.ErrRecordNotFound) {
		return apperror.Internal("Gagal memeriksa status tautan Google", errFind)
	}

	if errors.Is(errFind, gorm.ErrRecordNotFound) {
		newBind := &UserGoogleAccount{
			UserID:          userID,
			GoogleID:        sub,
			GoogleEmail:     email,
			GoogleName:      &name,
			GoogleAvatarURL: &picture,
			EmailVerified:   emailVerified,
		}
		if err := s.db.WithContext(ctx).Create(newBind).Error; err != nil {
			return apperror.Internal("Gagal menautkan akun Google", err)
		}
	} else {
		current.GoogleID = sub
		current.GoogleEmail = email
		current.GoogleName = &name
		current.GoogleAvatarURL = &picture
		current.EmailVerified = emailVerified
		if err := s.db.WithContext(ctx).Save(&current).Error; err != nil {
			return apperror.Internal("Gagal memperbarui tautan akun Google", err)
		}
	}

	return nil
}

func (s *service) GoogleUnbind(ctx context.Context, userID uuid.UUID) error {
	result := s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&UserGoogleAccount{})
	if result.Error != nil {
		return apperror.Internal("Gagal melepas tautan akun Google", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("Akun Google belum pernah ditautkan sebelumnya")
	}
	return nil
}

func (s *service) Refresh(ctx context.Context, rawRefreshToken string, clientIP, userAgent string) (*RefreshTokenResponse, string, error) {
	if rawRefreshToken == "" {
		return nil, "", apperror.Unauthorized("Sesi tidak ditemukan atau telah berakhir")
	}

	tokenHash := HashRefreshToken(rawRefreshToken, s.cfg.JWTSecret)

	var tokenRecord Token
	if err := s.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&tokenRecord).Error; err != nil {
		return nil, "", apperror.Unauthorized("Sesi login tidak valid atau telah kedaluwarsa")
	}

	// Revoke old token
	now := time.Now()
	s.db.WithContext(ctx).Model(&tokenRecord).Update("revoked_at", now)

	// Fetch user & role
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", tokenRecord.UserID).First(&user).Error; err != nil {
		return nil, "", apperror.Unauthorized("Pengguna sudah tidak aktif")
	}

	var role Role
	_ = s.db.WithContext(ctx).Where("id = ?", user.RoleID).First(&role)

	// Generate new access token & refresh token
	newAccessToken, expiresIn, err := GenerateAccessToken(&user, role.Name, s.cfg.JWTSecret)
	if err != nil {
		return nil, "", apperror.Internal("Gagal memperbarui access token", err)
	}

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", apperror.Internal("Gagal menerbitkan refresh token", err)
	}

	newTokenRecord := &Token{
		UserID:    user.ID,
		TokenHash: HashRefreshToken(newRefreshToken, s.cfg.JWTSecret),
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
		Remember:  tokenRecord.Remember,
		UserAgent: &userAgent,
		IPAddress: &clientIP,
	}
	if err := s.db.WithContext(ctx).Create(newTokenRecord).Error; err != nil {
		return nil, "", apperror.Internal("Gagal menyimpan sesi login baru", err)
	}

	return &RefreshTokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, newRefreshToken, nil
}

func (s *service) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken != "" {
		tokenHash := HashRefreshToken(rawRefreshToken, s.cfg.JWTSecret)
		now := time.Now()
		s.db.WithContext(ctx).Model(&Token{}).
			Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
			Update("revoked_at", now)
	}
	return nil
}

func (s *service) GetMe(ctx context.Context, userID uuid.UUID) (*UserProfileDTO, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil profil pengguna", err)
	}

	var role Role
	_ = s.db.WithContext(ctx).Where("id = ?", user.RoleID).First(&role)

	var googleCount int64
	s.db.WithContext(ctx).Model(&UserGoogleAccount{}).Where("user_id = ?", user.ID).Count(&googleCount)

	return &UserProfileDTO{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		AvatarURL:    user.AvatarURL,
		Status:       user.Status,
		RoleID:       user.RoleID,
		RoleName:     role.Name,
		DepartmentID: user.DepartmentID,
		CompanyID:    user.CompanyID,
		GoogleBound:  googleCount > 0,
	}, nil
}
