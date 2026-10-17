package asset

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/pagination"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	PresignUpload(ctx context.Context, actor *middleware.AuthClaims, req PresignAssetRequest) (*PresignAssetResponse, error)
	CreateAsset(ctx context.Context, actor *middleware.AuthClaims, req CreateAssetRequest) (*AssetResponse, error)
	GetAssets(ctx context.Context, actor *middleware.AuthClaims, filter AssetFilterQuery, p pagination.Params) ([]AssetResponse, pagination.Meta, error)
	GetAssetByID(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) (*AssetResponse, error)
	DeleteAsset(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error
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

var safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFilename(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, " ", "-")
	clean := safeFilenameRegex.ReplaceAllString(base, "")
	if clean == "" || clean == "." {
		return "asset_" + uuid.New().String()[:8]
	}
	return clean
}

func (s *service) PresignUpload(ctx context.Context, actor *middleware.AuthClaims, req PresignAssetRequest) (*PresignAssetResponse, error) {
	if s.s3Client == nil {
		return nil, apperror.Internal("Layanan penyimpanan storage belum diinisialisasi", errors.New("s3 client is nil"))
	}

	folder := strings.Trim(req.Folder, "/")
	if folder == "" {
		folder = "assets"
	}
	// Sanitize folder
	folder = strings.ToLower(folder)
	folder = safeFilenameRegex.ReplaceAllString(folder, "")
	if folder == "" {
		folder = "assets"
	}

	now := time.Now().UTC()
	cleanName := sanitizeFilename(req.Filename)
	uniquePrefix := uuid.New().String()[:8]
	key := fmt.Sprintf("%s/%d/%02d/%s_%s", folder, now.Year(), now.Month(), uniquePrefix, cleanName)

	uploadURL, err := s.s3Client.PresignPutObject(ctx, key, req.ContentType, req.Size, 15*time.Minute)
	if err != nil {
		return nil, apperror.Internal("Gagal membuat presigned upload URL", err)
	}

	publicURL := s.s3Client.GetPublicURL(key)

	return &PresignAssetResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
		Path:      key,
	}, nil
}

func (s *service) CreateAsset(ctx context.Context, actor *middleware.AuthClaims, req CreateAssetRequest) (*AssetResponse, error) {
	asset := Asset{
		ID:        uuid.New(),
		Path:      req.Path,
		Filename:  req.Filename,
		MimeType:  req.MimeType,
		Size:      req.Size,
		CreatedBy: actor.UserID,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(&asset).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan metadata aset", err)
	}

	return s.toResponse(asset), nil
}

func (s *service) GetAssets(ctx context.Context, actor *middleware.AuthClaims, filter AssetFilterQuery, p pagination.Params) ([]AssetResponse, pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&Asset{}).Where("deleted_at IS NULL")

	if filter.MimeType != "" {
		if strings.EqualFold(filter.MimeType, "image") {
			query = query.Where("mime_type LIKE ?", "image/%")
		} else {
			query = query.Where("mime_type = ?", filter.MimeType)
		}
	}

	if p.Search != "" {
		query = query.Where("filename ILIKE ?", p.SearchPattern())
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, pagination.Meta{}, apperror.Internal("Gagal menghitung jumlah aset", err)
	}

	var assets []Asset
	if err := query.Scopes(p.Scope("created_at", "filename", "size")).Find(&assets).Error; err != nil {
		return nil, pagination.Meta{}, apperror.Internal("Gagal mengambil daftar aset", err)
	}

	responses := make([]AssetResponse, len(assets))
	for i, item := range assets {
		responses[i] = *s.toResponse(item)
	}

	return responses, p.BuildMeta(total), nil
}

func (s *service) GetAssetByID(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) (*AssetResponse, error) {
	var asset Asset
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Aset tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil data aset", err)
	}

	return s.toResponse(asset), nil
}

func (s *service) DeleteAsset(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error {
	var asset Asset
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Aset tidak ditemukan")
		}
		return apperror.Internal("Gagal menemukan aset", err)
	}

	// Only admin or owner can delete
	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	if !isAdmin && asset.CreatedBy != actor.UserID {
		return apperror.Forbidden("Anda tidak memiliki izin untuk menghapus aset ini")
	}

	// Delete from storage if client exists
	if s.s3Client != nil {
		_ = s.s3Client.DeleteObject(ctx, asset.Path)
	}

	if err := s.db.WithContext(ctx).Delete(&asset).Error; err != nil {
		return apperror.Internal("Gagal menghapus aset dari database", err)
	}

	return nil
}

func (s *service) toResponse(asset Asset) *AssetResponse {
	publicURL := ""
	if s.s3Client != nil {
		publicURL = s.s3Client.GetPublicURL(asset.Path)
	} else {
		publicURL = "/" + asset.Path
	}

	return &AssetResponse{
		ID:        asset.ID,
		Path:      asset.Path,
		URL:       publicURL,
		Filename:  asset.Filename,
		MimeType:  asset.MimeType,
		Size:      asset.Size,
		CreatedBy: asset.CreatedBy,
		CreatedAt: asset.CreatedAt,
	}
}
