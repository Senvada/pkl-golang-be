package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/shared/apperror"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserProfileResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	GenerateAvatarPresignedURL(ctx context.Context, userID uuid.UUID, req PresignAvatarRequest) (*PresignAvatarResponse, error)
	DeleteAvatar(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error)
}

type service struct {
	db       *gorm.DB
	cfg      *config.Config
	s3Client *storage.S3Client
}

func NewService(db *gorm.DB, cfg *config.Config, s3Client *storage.S3Client) Service {
	return &service{
		db:       db,
		cfg:      cfg,
		s3Client: s3Client,
	}
}

func (s *service) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil profil pengguna", err)
	}

	var role Role
	_ = s.db.WithContext(ctx).Where("id = ?", user.RoleID).First(&role)

	var department *Department
	if user.DepartmentID != nil {
		var dept Department
		if err := s.db.WithContext(ctx).Where("id = ?", *user.DepartmentID).First(&dept).Error; err == nil {
			department = &dept
		}
	}

	var googleCount int64
	s.db.WithContext(ctx).Model(&UserGoogleAccount{}).Where("user_id = ?", user.ID).Count(&googleCount)

	return s.buildProfileResponse(&user, &role, department, googleCount > 0), nil
}

func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserProfileResponse, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan profil pengguna", err)
	}

	updates := make(map[string]any)

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		if len(trimmedName) < 2 {
			return nil, apperror.BadRequest("Nama pengguna minimal 2 karakter", nil)
		}
		updates["name"] = trimmedName
		user.Name = trimmedName
	}

	if req.AvatarURL != nil {
		newAvatar := strings.TrimSpace(*req.AvatarURL)
		// Clean up previous avatar if it was hosted in our S3 bucket and different from new
		if s.s3Client != nil && user.AvatarURL != nil && *user.AvatarURL != "" && *user.AvatarURL != newAvatar {
			_ = s.s3Client.DeleteObjectByURL(ctx, *user.AvatarURL)
		}

		if newAvatar == "" {
			updates["avatar_url"] = nil
			user.AvatarURL = nil
		} else {
			updates["avatar_url"] = newAvatar
			user.AvatarURL = &newAvatar
		}
	}

	if req.PasswordOld != nil && req.PasswordNew != nil && *req.PasswordOld != "" && *req.PasswordNew != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(*req.PasswordOld)); err != nil {
			return nil, apperror.BadRequest("Kata sandi saat ini tidak cocok", nil)
		}
		if len(*req.PasswordNew) < 6 {
			return nil, apperror.BadRequest("Kata sandi baru minimal 6 karakter", nil)
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.PasswordNew), 10)
		if err != nil {
			return nil, apperror.Internal("Gagal mengenkripsi kata sandi baru", err)
		}
		updates["password"] = string(hashedPassword)
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
			return nil, apperror.Internal("Gagal memperbarui profil pengguna", err)
		}
	}

	return s.GetProfile(ctx, userID)
}

func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Pengguna tidak ditemukan")
		}
		return apperror.Internal("Gagal menemukan data pengguna", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return apperror.BadRequest("Kata sandi saat ini tidak cocok", nil)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		return apperror.Internal("Gagal mengenkripsi kata sandi baru", err)
	}

	if err := s.db.WithContext(ctx).Model(&user).Updates(map[string]any{
		"password":   string(hashedPassword),
		"updated_at": time.Now(),
	}).Error; err != nil {
		return apperror.Internal("Gagal memperbarui kata sandi", err)
	}

	return nil
}

func (s *service) GenerateAvatarPresignedURL(ctx context.Context, userID uuid.UUID, req PresignAvatarRequest) (*PresignAvatarResponse, error) {
	if s.s3Client == nil {
		return nil, apperror.Internal("Layanan penyimpanan storage belum diinisialisasi", errors.New("s3 client is nil"))
	}

	var ext string
	switch req.ContentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	default:
		return nil, apperror.BadRequest("Format gambar tidak didukung. Harap gunakan JPEG, PNG, WEBP, atau GIF", nil)
	}

	// Key format: avatars/{userID}/{timestamp}_{random}.{ext}
	key := fmt.Sprintf("avatars/%s/%d_%s%s", userID.String(), time.Now().Unix(), uuid.New().String()[:8], ext)

	// Presign PUT URL with 5-minute lifetime
	uploadURL, err := s.s3Client.PresignPutObject(ctx, key, req.ContentType, req.FileSize, 5*time.Minute)
	if err != nil {
		return nil, apperror.Internal("Gagal membuat presigned upload URL", err)
	}

	publicURL := s.s3Client.GetPublicURL(key)

	return &PresignAvatarResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
		Key:       key,
	}, nil
}

func (s *service) DeleteAvatar(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan profil pengguna", err)
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		if s.s3Client != nil {
			_ = s.s3Client.DeleteObjectByURL(ctx, *user.AvatarURL)
		}
		if err := s.db.WithContext(ctx).Model(&user).Updates(map[string]any{
			"avatar_url": nil,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return nil, apperror.Internal("Gagal menghapus foto profil", err)
		}
		user.AvatarURL = nil
	}

	return s.GetProfile(ctx, userID)
}

func (s *service) buildProfileResponse(user *User, role *Role, dept *Department, googleBound bool) *UserProfileResponse {
	resp := &UserProfileResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		AvatarURL:    user.AvatarURL,
		Image:        user.AvatarURL, // Alias for backward compatibility
		Status:       user.Status,
		RoleID:       user.RoleID,
		RoleName:     role.Name,
		Role:         &RoleDTO{ID: role.ID, Name: role.Name},
		DepartmentID: user.DepartmentID,
		CompanyID:    user.CompanyID,
		GoogleBound:  googleBound,
	}

	if dept != nil {
		resp.Department = &DepartmentDTO{
			ID:    dept.ID,
			Name:  dept.Name,
			Color: dept.Color,
		}
	}

	return resp
}
