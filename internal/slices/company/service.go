package company

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/shared/apperror"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context) ([]CompanyResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CompanyResponse, error)
	GetBySlug(ctx context.Context, slug string) (*CompanyResponse, error)
	Create(ctx context.Context, creatorID uuid.UUID, req CreateCompanyRequest) (*CompanyResponse, error)
	Update(ctx context.Context, updaterID uuid.UUID, id uuid.UUID, req UpdateCompanyRequest) (*CompanyResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
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

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)

func generateSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphaNumRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "company-" + uuid.New().String()[:8]
	}
	return s
}

func (s *service) GetAll(ctx context.Context) ([]CompanyResponse, error) {
	var companies []Company
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Find(&companies).Error; err != nil {
		return nil, apperror.Internal("Gagal mengambil data perusahaan mitra", err)
	}

	type companyMemberCount struct {
		CompanyID uuid.UUID `gorm:"column:company_id"`
		Count     int64     `gorm:"column:count"`
	}
	var counts []companyMemberCount
	_ = s.db.WithContext(ctx).
		Table("users").
		Select("company_id, count(*) as count").
		Where("deleted_at IS NULL AND company_id IS NOT NULL").
		Group("company_id").
		Scan(&counts)

	countMap := make(map[uuid.UUID]int64, len(counts))
	for _, c := range counts {
		countMap[c.CompanyID] = c.Count
	}

	responses := make([]CompanyResponse, len(companies))
	for i, c := range companies {
		responses[i] = CompanyResponse{
			ID:          c.ID,
			Name:        c.Name,
			Slug:        c.Slug,
			Email:       c.Email,
			PhoneNumber: c.PhoneNumber,
			Address:     c.Address,
			Description: c.Description,
			Content:     c.Content,
			ImageURL:    c.ImageURL,
			GmapsLink:   c.GmapsLink,
			Lat:         c.Lat,
			Lng:         c.Lng,
			Status:      c.Status,
			MemberCount: countMap[c.ID],
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		}
	}

	return responses, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil detail perusahaan", err)
	}

	var memberCount int64
	_ = s.db.WithContext(ctx).
		Table("users").
		Where("company_id = ? AND deleted_at IS NULL", id).
		Count(&memberCount)

	return &CompanyResponse{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		Email:       c.Email,
		PhoneNumber: c.PhoneNumber,
		Address:     c.Address,
		Description: c.Description,
		Content:     c.Content,
		ImageURL:    c.ImageURL,
		GmapsLink:   c.GmapsLink,
		Lat:         c.Lat,
		Lng:         c.Lng,
		Status:      c.Status,
		MemberCount: memberCount,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}, nil
}

func (s *service) GetBySlug(ctx context.Context, slug string) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil detail perusahaan", err)
	}

	var memberCount int64
	_ = s.db.WithContext(ctx).
		Table("users").
		Where("company_id = ? AND deleted_at IS NULL", c.ID).
		Count(&memberCount)

	return &CompanyResponse{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		Email:       c.Email,
		PhoneNumber: c.PhoneNumber,
		Address:     c.Address,
		Description: c.Description,
		Content:     c.Content,
		ImageURL:    c.ImageURL,
		GmapsLink:   c.GmapsLink,
		Lat:         c.Lat,
		Lng:         c.Lng,
		Status:      c.Status,
		MemberCount: memberCount,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}, nil
}

func (s *service) Create(ctx context.Context, creatorID uuid.UUID, req CreateCompanyRequest) (*CompanyResponse, error) {
	baseSlug := generateSlug(req.Name)
	slug := baseSlug

	// Ensure unique slug
	for i := 1; ; i++ {
		var count int64
		s.db.WithContext(ctx).Model(&Company{}).Where("slug = ? AND deleted_at IS NULL", slug).Count(&count)
		if count == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	var content datatypes.JSON
	if req.Content != nil {
		content = *req.Content
	} else {
		content = datatypes.JSON([]byte("{}"))
	}

	c := Company{
		Name:        strings.TrimSpace(req.Name),
		Slug:        slug,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
		Description: req.Description,
		Content:     content,
		ImageURL:    req.ImageURL,
		GmapsLink:   req.GmapsLink,
		Lat:         req.Lat,
		Lng:         req.Lng,
		Status:      status,
		CreatedBy:   &creatorID,
		UpdatedBy:   &creatorID,
	}

	if err := s.db.WithContext(ctx).Create(&c).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan data perusahaan mitra", err)
	}

	return s.GetByID(ctx, c.ID)
}

func (s *service) Update(ctx context.Context, updaterID uuid.UUID, id uuid.UUID, req UpdateCompanyRequest) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan perusahaan", err)
	}

	updates := make(map[string]any)

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		updates["name"] = trimmedName
		c.Name = trimmedName
	}
	if req.Email != nil {
		updates["email"] = req.Email
	}
	if req.PhoneNumber != nil {
		updates["phone_number"] = req.PhoneNumber
	}
	if req.Address != nil {
		updates["address"] = req.Address
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.ImageURL != nil {
		updates["image_url"] = req.ImageURL
	}
	if req.GmapsLink != nil {
		updates["gmaps_link"] = req.GmapsLink
	}
	if req.Lat != nil {
		updates["lat"] = req.Lat
	}
	if req.Lng != nil {
		updates["lng"] = req.Lng
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		updates["updated_by"] = &updaterID
		updates["updated_at"] = time.Now()
		if err := s.db.WithContext(ctx).Model(&c).Updates(updates).Error; err != nil {
			return nil, apperror.Internal("Gagal memperbarui data perusahaan", err)
		}
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Perusahaan tidak ditemukan")
		}
		return apperror.Internal("Gagal menemukan perusahaan", err)
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&c).Update("deleted_at", now).Error; err != nil {
		return apperror.Internal("Gagal menghapus perusahaan", err)
	}

	return nil
}
