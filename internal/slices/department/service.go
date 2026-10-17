package department

import (
	"context"
	"errors"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/shared/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context) ([]DepartmentResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*DepartmentResponse, error)
	Create(ctx context.Context, req CreateDepartmentRequest) (*DepartmentResponse, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateDepartmentRequest) (*DepartmentResponse, error)
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

func normalizeHexColor(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "#10B981"
	}
	if !strings.HasPrefix(c, "#") {
		c = "#" + c
	}
	return strings.ToUpper(c)
}

func (s *service) GetAll(ctx context.Context) ([]DepartmentResponse, error) {
	var depts []Department
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("code ASC").
		Find(&depts).Error; err != nil {
		return nil, apperror.Internal("Gagal mengambil daftar jurusan", err)
	}

	type deptUserCount struct {
		DepartmentID uuid.UUID `gorm:"column:department_id"`
		Count        int64     `gorm:"column:count"`
	}
	var counts []deptUserCount
	_ = s.db.WithContext(ctx).
		Table("users").
		Select("department_id, count(*) as count").
		Where("deleted_at IS NULL AND department_id IS NOT NULL").
		Group("department_id").
		Scan(&counts)

	countMap := make(map[uuid.UUID]int64, len(counts))
	for _, c := range counts {
		countMap[c.DepartmentID] = c.Count
	}

	responses := make([]DepartmentResponse, len(depts))
	for i, d := range depts {
		responses[i] = DepartmentResponse{
			ID:        d.ID,
			Name:      d.Name,
			Code:      d.Code,
			Color:     d.Color,
			UserCount: countMap[d.ID],
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}

	return responses, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*DepartmentResponse, error) {
	var dept Department
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Jurusan tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil detail jurusan", err)
	}

	var userCount int64
	_ = s.db.WithContext(ctx).
		Table("users").
		Where("department_id = ? AND deleted_at IS NULL", id).
		Count(&userCount).Error

	return &DepartmentResponse{
		ID:        dept.ID,
		Name:      dept.Name,
		Code:      dept.Code,
		Color:     dept.Color,
		UserCount: userCount,
		CreatedAt: dept.CreatedAt,
		UpdatedAt: dept.UpdatedAt,
	}, nil
}

func (s *service) Create(ctx context.Context, req CreateDepartmentRequest) (*DepartmentResponse, error) {
	trimmedCode := strings.ToUpper(strings.TrimSpace(req.Code))
	trimmedName := strings.TrimSpace(req.Name)

	var existing Department
	if err := s.db.WithContext(ctx).
		Where("UPPER(code) = ? AND deleted_at IS NULL", trimmedCode).
		First(&existing).Error; err == nil {
		return nil, apperror.Conflict("Kode jurusan '" + trimmedCode + "' sudah digunakan")
	}

	dept := Department{
		Name:  trimmedName,
		Code:  trimmedCode,
		Color: normalizeHexColor(req.Color),
	}

	if err := s.db.WithContext(ctx).Create(&dept).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan jurusan baru", err)
	}

	return &DepartmentResponse{
		ID:        dept.ID,
		Name:      dept.Name,
		Code:      dept.Code,
		Color:     dept.Color,
		UserCount: 0,
		CreatedAt: dept.CreatedAt,
		UpdatedAt: dept.UpdatedAt,
	}, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateDepartmentRequest) (*DepartmentResponse, error) {
	var dept Department
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Jurusan tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan jurusan", err)
	}

	updates := make(map[string]any)

	if req.Code != nil {
		trimmedCode := strings.ToUpper(strings.TrimSpace(*req.Code))
		if trimmedCode != dept.Code {
			var existing Department
			if err := s.db.WithContext(ctx).
				Where("UPPER(code) = ? AND id != ? AND deleted_at IS NULL", trimmedCode, id).
				First(&existing).Error; err == nil {
				return nil, apperror.Conflict("Kode jurusan '" + trimmedCode + "' sudah digunakan")
			}
			updates["code"] = trimmedCode
			dept.Code = trimmedCode
		}
	}

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		updates["name"] = trimmedName
		dept.Name = trimmedName
	}

	if req.Color != nil {
		normColor := normalizeHexColor(*req.Color)
		updates["color"] = normColor
		dept.Color = normColor
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.db.WithContext(ctx).Model(&dept).Updates(updates).Error; err != nil {
			return nil, apperror.Internal("Gagal memperbarui data jurusan", err)
		}
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	var dept Department
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Jurusan tidak ditemukan")
		}
		return apperror.Internal("Gagal menemukan jurusan", err)
	}

	var userCount int64
	if err := s.db.WithContext(ctx).
		Table("users").
		Where("department_id = ? AND deleted_at IS NULL", id).
		Count(&userCount).Error; err != nil {
		return apperror.Internal("Gagal memeriksa keterikatan pengguna pada jurusan", err)
	}

	if userCount > 0 {
		return apperror.BadRequest("Jurusan tidak dapat dihapus karena masih digunakan oleh pengguna aktif", nil)
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&dept).Update("deleted_at", now).Error; err != nil {
		return apperror.Internal("Gagal menghapus data jurusan", err)
	}

	return nil
}
