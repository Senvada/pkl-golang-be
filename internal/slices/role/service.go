package role

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/authutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context) ([]RoleResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*RoleResponse, error)
	Create(ctx context.Context, req CreateRoleRequest) (*RoleResponse, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*RoleResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CheckPermission(ctx context.Context, roleID uuid.UUID, module string, action string) (bool, error)
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
	if !strings.HasPrefix(c, "#") {
		c = "#" + c
	}
	return strings.ToUpper(c)
}

func (s *service) GetAll(ctx context.Context) ([]RoleResponse, error) {
	var roles []Role
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at ASC").
		Find(&roles).Error; err != nil {
		return nil, apperror.Internal("Gagal mengambil daftar role", err)
	}

	// Fetch active user counts grouped by role_id
	type roleUserCount struct {
		RoleID uuid.UUID `gorm:"column:role_id"`
		Count  int64     `gorm:"column:count"`
	}
	var counts []roleUserCount
	_ = s.db.WithContext(ctx).
		Table("users").
		Select("role_id, count(*) as count").
		Where("deleted_at IS NULL").
		Group("role_id").
		Scan(&counts)

	countMap := make(map[uuid.UUID]int64, len(counts))
	for _, c := range counts {
		countMap[c.RoleID] = c.Count
	}

	responses := make([]RoleResponse, len(roles))
	for i, r := range roles {
		responses[i] = RoleResponse{
			ID:          r.ID,
			Name:        r.Name,
			Color:       r.Color,
			Permissions: r.Permissions,
			UserCount:   countMap[r.ID],
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		}
	}

	return responses, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*RoleResponse, error) {
	var role Role
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Role tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil detail role", err)
	}

	var userCount int64
	_ = s.db.WithContext(ctx).
		Table("users").
		Where("role_id = ? AND deleted_at IS NULL", id).
		Count(&userCount)

	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Color:       role.Color,
		Permissions: role.Permissions,
		UserCount:   userCount,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}, nil
}

func (s *service) Create(ctx context.Context, req CreateRoleRequest) (*RoleResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperror.Validation("Nama role tidak boleh kosong", nil)
	}

	// Check for unique role name among active roles
	var existing Role
	err := s.db.WithContext(ctx).
		Where("LOWER(name) = ? AND deleted_at IS NULL", strings.ToLower(name)).
		First(&existing).Error
	if err == nil {
		return nil, apperror.Conflict("Nama role sudah digunakan")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Internal("Gagal memeriksa keunikan nama role", err)
	}

	// Validate permissions JSON
	if len(req.Permissions) > 0 && !json.Valid(req.Permissions) {
		return nil, apperror.Validation("Format permissions harus berupa JSON yang valid", nil)
	}

	role := Role{
		Name:        name,
		Color:       normalizeHexColor(req.Color),
		Permissions: req.Permissions,
	}

	if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan data role baru", err)
	}

	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Color:       role.Color,
		Permissions: role.Permissions,
		UserCount:   0,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*RoleResponse, error) {
	var role Role
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Role tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mencari data role", err)
	}

	// If name update requested
	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			return nil, apperror.Validation("Nama role tidak boleh kosong", nil)
		}

		// Prevent renaming the core system 'Admin' role
		if strings.EqualFold(role.Name, "Admin") && !strings.EqualFold(newName, "Admin") {
			return nil, apperror.Forbidden("Role 'Admin' adalah role sistem utama dan tidak dapat diganti namanya")
		}

		// Check name uniqueness if changed
		if !strings.EqualFold(role.Name, newName) {
			var existing Role
			err := s.db.WithContext(ctx).
				Where("LOWER(name) = ? AND id != ? AND deleted_at IS NULL", strings.ToLower(newName), id).
				First(&existing).Error
			if err == nil {
				return nil, apperror.Conflict("Nama role sudah digunakan")
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.Internal("Gagal memeriksa keunikan nama role", err)
			}
		}
		role.Name = newName
	}

	// If color update requested
	if req.Color != nil && strings.TrimSpace(*req.Color) != "" {
		role.Color = normalizeHexColor(*req.Color)
	}

	// If permissions update requested
	if req.Permissions != nil {
		if !json.Valid(req.Permissions) {
			return nil, apperror.Validation("Format permissions harus berupa JSON yang valid", nil)
		}
		role.Permissions = req.Permissions
	}

	if err := s.db.WithContext(ctx).Save(&role).Error; err != nil {
		return nil, apperror.Internal("Gagal memperbarui data role", err)
	}

	var userCount int64
	_ = s.db.WithContext(ctx).
		Table("users").
		Where("role_id = ? AND deleted_at IS NULL", id).
		Count(&userCount)

	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Color:       role.Color,
		Permissions: role.Permissions,
		UserCount:   userCount,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	var role Role
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Role tidak ditemukan")
		}
		return apperror.Internal("Gagal mencari data role", err)
	}

	// Prevent deleting the primary Admin role
	if strings.EqualFold(role.Name, "Admin") {
		return apperror.Forbidden("Role 'Admin' adalah role sistem utama dan tidak dapat dihapus")
	}

	// Check if active users are still assigned to this role
	var activeUserCount int64
	if err := s.db.WithContext(ctx).
		Table("users").
		Where("role_id = ? AND deleted_at IS NULL", id).
		Count(&activeUserCount).Error; err != nil {
		return apperror.Internal("Gagal memvalidasi keterkaitan pengguna", err)
	}

	if activeUserCount > 0 {
		return apperror.Conflict(fmt.Sprintf("Role '%s' tidak dapat dihapus karena masih digunakan oleh %d pengguna aktif", role.Name, activeUserCount))
	}

	if err := s.db.WithContext(ctx).Delete(&role).Error; err != nil {
		return apperror.Internal("Gagal menghapus data role", err)
	}

	return nil
}

func (s *service) CheckPermission(ctx context.Context, roleID uuid.UUID, module string, action string) (bool, error) {
	var role Role
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", roleID).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return authutil.CheckRolePermission(ctx, s.db, role.ID, role.Name, module, action)
}
