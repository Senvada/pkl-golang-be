package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/pagination"

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

	// Management operations
	GetUsers(ctx context.Context, actor *middleware.AuthClaims, filter UserFilterQuery, p pagination.Params) ([]UserItemResponse, pagination.Meta, error)
	GetUserByID(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) (*UserItemResponse, error)
	CreateUser(ctx context.Context, actor *middleware.AuthClaims, req CreateUserRequest) (*UserItemResponse, error)
	UpdateUser(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateUserRequest) (*UserItemResponse, error)
	DeleteUser(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error
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

	var comp *Company
	if user.CompanyID != nil {
		var c Company
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *user.CompanyID).First(&c).Error; err == nil {
			comp = &c
		}
	}

	var parentName *string
	if user.ParentID != nil {
		var parent User
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *user.ParentID).First(&parent).Error; err == nil {
			parentName = &parent.Name
		}
	}

	var googleCount int64
	s.db.WithContext(ctx).Model(&UserGoogleAccount{}).Where("user_id = ?", user.ID).Count(&googleCount)

	return s.buildProfileResponse(&user, &role, department, comp, parentName, googleCount > 0), nil
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

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return apperror.BadRequest("Kata sandi saat ini tidak cocok", nil)
	}

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

	key := fmt.Sprintf("avatars/%s/avatar_%d_%s%s", userID.String(), time.Now().Unix(), uuid.New().String()[:6], ext)

	uploadURL, err := s.s3Client.PresignPutObject(ctx, key, req.ContentType, 0, 5*time.Minute)
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

// -----------------------------------------------------------------------
// USER MANAGEMENT OPERATIONS
// -----------------------------------------------------------------------

func (s *service) GetUsers(ctx context.Context, actor *middleware.AuthClaims, filter UserFilterQuery, p pagination.Params) ([]UserItemResponse, pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&User{}).Where("users.deleted_at IS NULL")

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isIndustri := strings.EqualFold(actor.RoleName, "industri") || strings.EqualFold(actor.RoleName, "pembimbing industri")

	// Scoping logic:
	// - If Industri / Company PIC: Restricted to users under their company OR supervised by them (parent_id)
	if !isAdmin && isIndustri {
		if actor.CompanyID != nil {
			query = query.Where("(users.company_id = ? OR users.parent_id = ?)", *actor.CompanyID, actor.UserID)
		} else {
			query = query.Where("users.parent_id = ?", actor.UserID)
		}
	}

	// Filter Search
	if filter.Search != "" {
		pat := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("(LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?)", pat, pat)
	}

	// Filter Role
	if filter.RoleID != nil {
		query = query.Where("users.role_id = ?", *filter.RoleID)
	}

	// Filter Department (Jurusan)
	if filter.DepartmentID != nil {
		query = query.Where("users.department_id = ?", *filter.DepartmentID)
	}

	// Filter Company (Industri)
	if filter.CompanyID != nil {
		query = query.Where("users.company_id = ?", *filter.CompanyID)
	}

	// Filter Parent (PIC)
	if filter.ParentID != nil {
		query = query.Where("users.parent_id = ?", *filter.ParentID)
	}

	// Filter Status
	if filter.Status != "" {
		query = query.Where("users.status = ?", filter.Status)
	}

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, pagination.Meta{}, apperror.Internal("Gagal menghitung jumlah pengguna", err)
	}

	meta := p.BuildMeta(totalRows)

	var users []User
	if err := query.Scopes(p.Scope("created_at", "name", "email", "status")).Find(&users).Error; err != nil {
		return nil, meta, apperror.Internal("Gagal mengambil data daftar pengguna", err)
	}

	if len(users) == 0 {
		return []UserItemResponse{}, meta, nil
	}

	// Batch gather IDs for eager lookup
	roleIDs := make([]uuid.UUID, 0, len(users))
	deptIDs := make([]uuid.UUID, 0, len(users))
	compIDs := make([]uuid.UUID, 0, len(users))
	parentIDs := make([]uuid.UUID, 0, len(users))
	userIDs := make([]uuid.UUID, len(users))

	for i, u := range users {
		userIDs[i] = u.ID
		roleIDs = append(roleIDs, u.RoleID)
		if u.DepartmentID != nil {
			deptIDs = append(deptIDs, *u.DepartmentID)
		}
		if u.CompanyID != nil {
			compIDs = append(compIDs, *u.CompanyID)
		}
		if u.ParentID != nil {
			parentIDs = append(parentIDs, *u.ParentID)
		}
	}

	// Query Roles
	roleMap := make(map[uuid.UUID]Role)
	if len(roleIDs) > 0 {
		var roles []Role
		s.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&roles)
		for _, r := range roles {
			roleMap[r.ID] = r
		}
	}

	// Query Departments
	deptMap := make(map[uuid.UUID]Department)
	if len(deptIDs) > 0 {
		var depts []Department
		s.db.WithContext(ctx).Where("id IN ?", deptIDs).Find(&depts)
		for _, d := range depts {
			deptMap[d.ID] = d
		}
	}

	// Query Companies
	compMap := make(map[uuid.UUID]Company)
	if len(compIDs) > 0 {
		var comps []Company
		s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", compIDs).Find(&comps)
		for _, c := range comps {
			compMap[c.ID] = c
		}
	}

	// Query Parents
	parentMap := make(map[uuid.UUID]string)
	if len(parentIDs) > 0 {
		var parents []User
		s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", parentIDs).Find(&parents)
		for _, pUser := range parents {
			parentMap[pUser.ID] = pUser.Name
		}
	}

	// Query SubUser counts (how many users have parent_id = user.ID)
	type subCount struct {
		ParentID uuid.UUID `gorm:"column:parent_id"`
		Count    int64     `gorm:"column:count"`
	}
	var subCounts []subCount
	s.db.WithContext(ctx).
		Table("users").
		Select("parent_id, count(*) as count").
		Where("parent_id IN ? AND deleted_at IS NULL", userIDs).
		Group("parent_id").
		Scan(&subCounts)

	subCountMap := make(map[uuid.UUID]int64, len(subCounts))
	for _, sc := range subCounts {
		subCountMap[sc.ParentID] = sc.Count
	}

	// Query Google accounts
	type googleRow struct {
		UserID uuid.UUID `gorm:"column:user_id"`
	}
	var googleRows []googleRow
	s.db.WithContext(ctx).
		Table("user_google_accounts").
		Select("user_id").
		Where("user_id IN ?", userIDs).
		Scan(&googleRows)

	googleMap := make(map[uuid.UUID]bool, len(googleRows))
	for _, gr := range googleRows {
		googleMap[gr.UserID] = true
	}

	// Assemble responses
	items := make([]UserItemResponse, len(users))
	for i, u := range users {
		r := roleMap[u.RoleID]
		item := UserItemResponse{
			ID:           u.ID,
			Name:         u.Name,
			Email:        u.Email,
			AvatarURL:    u.AvatarURL,
			Image:        u.AvatarURL,
			Status:       u.Status,
			RoleID:       u.RoleID,
			RoleName:     r.Name,
			Role:         &RoleDTO{ID: r.ID, Name: r.Name, Color: r.Color},
			DepartmentID: u.DepartmentID,
			CompanyID:    u.CompanyID,
			ParentID:     u.ParentID,
			SubUserCount: subCountMap[u.ID],
			GoogleBound:  googleMap[u.ID],
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
		}

		if u.DepartmentID != nil {
			if d, ok := deptMap[*u.DepartmentID]; ok {
				item.DepartmentName = &d.Name
				item.Department = &DepartmentDTO{
					ID:    d.ID,
					Name:  d.Name,
					Code:  d.Code,
					Color: d.Color,
				}
			}
		}

		if u.CompanyID != nil {
			if c, ok := compMap[*u.CompanyID]; ok {
				item.CompanyName = &c.Name
				item.Company = &CompanyDTO{
					ID:   c.ID,
					Name: c.Name,
					Slug: c.Slug,
				}
			}
		}

		if u.ParentID != nil {
			if pName, ok := parentMap[*u.ParentID]; ok {
				item.ParentName = &pName
			}
		}

		items[i] = item
	}

	return items, meta, nil
}

func (s *service) GetUserByID(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) (*UserItemResponse, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil data pengguna", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isIndustri := strings.EqualFold(actor.RoleName, "industri") || strings.EqualFold(actor.RoleName, "pembimbing industri")

	if !isAdmin && isIndustri {
		allowed := false
		if actor.CompanyID != nil && user.CompanyID != nil && *user.CompanyID == *actor.CompanyID {
			allowed = true
		}
		if user.ParentID != nil && *user.ParentID == actor.UserID {
			allowed = true
		}
		if user.ID == actor.UserID {
			allowed = true
		}
		if !allowed {
			return nil, apperror.Forbidden("Anda tidak memiliki izin untuk melihat pengguna ini")
		}
	}

	var role Role
	_ = s.db.WithContext(ctx).Where("id = ?", user.RoleID).First(&role)

	var department *Department
	if user.DepartmentID != nil {
		var d Department
		if err := s.db.WithContext(ctx).Where("id = ?", *user.DepartmentID).First(&d).Error; err == nil {
			department = &d
		}
	}

	var comp *Company
	if user.CompanyID != nil {
		var c Company
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *user.CompanyID).First(&c).Error; err == nil {
			comp = &c
		}
	}

	var parentName *string
	if user.ParentID != nil {
		var pUser User
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *user.ParentID).First(&pUser).Error; err == nil {
			parentName = &pUser.Name
		}
	}

	var subUserCount int64
	s.db.WithContext(ctx).Table("users").Where("parent_id = ? AND deleted_at IS NULL", user.ID).Count(&subUserCount)

	var googleCount int64
	s.db.WithContext(ctx).Model(&UserGoogleAccount{}).Where("user_id = ?", user.ID).Count(&googleCount)

	item := &UserItemResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		AvatarURL:    user.AvatarURL,
		Image:        user.AvatarURL,
		Status:       user.Status,
		RoleID:       user.RoleID,
		RoleName:     role.Name,
		Role:         &RoleDTO{ID: role.ID, Name: role.Name, Color: role.Color},
		DepartmentID: user.DepartmentID,
		CompanyID:    user.CompanyID,
		ParentID:     user.ParentID,
		ParentName:   parentName,
		SubUserCount: subUserCount,
		GoogleBound:  googleCount > 0,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	if department != nil {
		item.DepartmentName = &department.Name
		item.Department = &DepartmentDTO{
			ID:    department.ID,
			Name:  department.Name,
			Code:  department.Code,
			Color: department.Color,
		}
	}

	if comp != nil {
		item.CompanyName = &comp.Name
		item.Company = &CompanyDTO{
			ID:   comp.ID,
			Name: comp.Name,
			Slug: comp.Slug,
		}
	}

	return item, nil
}

func (s *service) CreateUser(ctx context.Context, actor *middleware.AuthClaims, req CreateUserRequest) (*UserItemResponse, error) {
	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isIndustri := strings.EqualFold(actor.RoleName, "industri") || strings.EqualFold(actor.RoleName, "pembimbing industri")

	trimmedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	trimmedName := strings.TrimSpace(req.Name)

	// Check email uniqueness
	var count int64
	s.db.WithContext(ctx).Model(&User{}).Where("LOWER(email) = ? AND deleted_at IS NULL", trimmedEmail).Count(&count)
	if count > 0 {
		return nil, apperror.Conflict("Alamat email '" + trimmedEmail + "' sudah terdaftar")
	}

	// Verify target role exists
	var targetRole Role
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.RoleID).First(&targetRole).Error; err != nil {
		return nil, apperror.BadRequest("Role yang dipilih tidak valid atau tidak ditemukan", nil)
	}

	// Non-admin users cannot create "Admin" accounts
	if strings.EqualFold(targetRole.Name, "admin") && !isAdmin {
		return nil, apperror.Forbidden("Hanya administrator yang dapat membuat akun dengan role Admin")
	}

	finalCompanyID := req.CompanyID
	finalParentID := req.ParentID

	// If actor is Industry Lead: automatically lock company_id to their company and parent_id to themselves
	if !isAdmin && isIndustri {
		if actor.CompanyID == nil {
			return nil, apperror.Forbidden("Akun industri Anda belum ditautkan ke perusahaan mitra")
		}
		finalCompanyID = actor.CompanyID
		finalParentID = &actor.UserID
	}

	// Verify department if provided
	if req.DepartmentID != nil {
		var deptCount int64
		s.db.WithContext(ctx).Model(&Department{}).Where("id = ? AND deleted_at IS NULL", *req.DepartmentID).Count(&deptCount)
		if deptCount == 0 {
			return nil, apperror.BadRequest("Jurusan yang dipilih tidak valid atau tidak ditemukan", nil)
		}
	}

	// Verify company if provided
	if finalCompanyID != nil {
		var compCount int64
		s.db.WithContext(ctx).Model(&Company{}).Where("id = ? AND deleted_at IS NULL", *finalCompanyID).Count(&compCount)
		if compCount == 0 {
			return nil, apperror.BadRequest("Perusahaan mitra yang dipilih tidak ditemukan", nil)
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, apperror.Internal("Gagal mengenkripsi kata sandi pengguna", err)
	}

	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	user := User{
		RoleID:       req.RoleID,
		DepartmentID: req.DepartmentID,
		CompanyID:    finalCompanyID,
		ParentID:     finalParentID,
		Name:         trimmedName,
		Email:        trimmedEmail,
		Password:     string(hashedPassword),
		Status:       status,
		CreatedBy:    &actor.UserID,
		UpdatedBy:    &actor.UserID,
	}

	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan data pengguna baru", err)
	}

	return s.GetUserByID(ctx, actor, user.ID)
}

func (s *service) UpdateUser(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateUserRequest) (*UserItemResponse, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan pengguna", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isIndustri := strings.EqualFold(actor.RoleName, "industri") || strings.EqualFold(actor.RoleName, "pembimbing industri")

	// Scoping check
	if !isAdmin {
		allowed := false
		if isIndustri {
			if actor.CompanyID != nil && user.CompanyID != nil && *user.CompanyID == *actor.CompanyID {
				allowed = true
			}
			if user.ParentID != nil && *user.ParentID == actor.UserID {
				allowed = true
			}
		}
		if user.ID == actor.UserID {
			allowed = true
		}
		if !allowed {
			return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengubah data pengguna ini")
		}
	}

	updates := make(map[string]any)

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		updates["name"] = trimmedName
		user.Name = trimmedName
	}

	if req.Email != nil {
		trimmedEmail := strings.ToLower(strings.TrimSpace(*req.Email))
		if trimmedEmail != strings.ToLower(user.Email) {
			var count int64
			s.db.WithContext(ctx).Model(&User{}).Where("LOWER(email) = ? AND id != ? AND deleted_at IS NULL", trimmedEmail, id).Count(&count)
			if count > 0 {
				return nil, apperror.Conflict("Alamat email '" + trimmedEmail + "' sudah digunakan oleh akun lain")
			}
			updates["email"] = trimmedEmail
			user.Email = trimmedEmail
		}
	}

	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 10)
		if err != nil {
			return nil, apperror.Internal("Gagal mengenkripsi kata sandi baru", err)
		}
		updates["password"] = string(hashed)
	}

	if req.RoleID != nil {
		if !isAdmin {
			return nil, apperror.Forbidden("Hanya administrator yang dapat mengubah role pengguna")
		}
		var roleCount int64
		s.db.WithContext(ctx).Model(&Role{}).Where("id = ? AND deleted_at IS NULL", *req.RoleID).Count(&roleCount)
		if roleCount == 0 {
			return nil, apperror.BadRequest("Role yang dipilih tidak valid", nil)
		}
		updates["role_id"] = *req.RoleID
	}

	if req.DepartmentID != nil {
		var deptCount int64
		s.db.WithContext(ctx).Model(&Department{}).Where("id = ? AND deleted_at IS NULL", *req.DepartmentID).Count(&deptCount)
		if deptCount == 0 {
			return nil, apperror.BadRequest("Jurusan yang dipilih tidak valid", nil)
		}
		updates["department_id"] = *req.DepartmentID
	}

	if req.CompanyID != nil {
		if !isAdmin {
			return nil, apperror.Forbidden("Hanya administrator yang dapat memindahkan perusahaan mitra pengguna")
		}
		var compCount int64
		s.db.WithContext(ctx).Model(&Company{}).Where("id = ? AND deleted_at IS NULL", *req.CompanyID).Count(&compCount)
		if compCount == 0 {
			return nil, apperror.BadRequest("Perusahaan mitra tidak ditemukan", nil)
		}
		updates["company_id"] = *req.CompanyID
	}

	if req.ParentID != nil && isAdmin {
		updates["parent_id"] = *req.ParentID
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	var oldAvatarToDelete string
	if req.AvatarURL != nil {
		newAvatar := strings.TrimSpace(*req.AvatarURL)
		if s.s3Client != nil && user.AvatarURL != nil && *user.AvatarURL != "" && *user.AvatarURL != newAvatar {
			oldAvatarToDelete = *user.AvatarURL
		}
		if newAvatar == "" {
			updates["avatar_url"] = nil
			user.AvatarURL = nil
		} else {
			updates["avatar_url"] = &newAvatar
			user.AvatarURL = &newAvatar
		}
	}

	if len(updates) > 0 {
		updates["updated_by"] = &actor.UserID
		updates["updated_at"] = time.Now()
		if err := s.db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
			return nil, apperror.Internal("Gagal memperbarui data pengguna", err)
		}
	}

	if oldAvatarToDelete != "" && s.s3Client != nil {
		_ = s.s3Client.DeleteObjectByURL(ctx, oldAvatarToDelete)
	}

	return s.GetUserByID(ctx, actor, id)
}

func (s *service) DeleteUser(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error {
	if actor.UserID == id {
		return apperror.BadRequest("Anda tidak dapat menghapus akun Anda sendiri", nil)
	}

	var user User
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Pengguna tidak ditemukan")
		}
		return apperror.Internal("Gagal menemukan pengguna", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isIndustri := strings.EqualFold(actor.RoleName, "industri") || strings.EqualFold(actor.RoleName, "pembimbing industri")

	if !isAdmin {
		allowed := false
		if isIndustri {
			if actor.CompanyID != nil && user.CompanyID != nil && *user.CompanyID == *actor.CompanyID {
				allowed = true
			}
			if user.ParentID != nil && *user.ParentID == actor.UserID {
				allowed = true
			}
		}
		if !allowed {
			return apperror.Forbidden("Anda tidak memiliki izin untuk menghapus pengguna ini")
		}
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&user).Updates(map[string]any{
		"deleted_at": now,
		"updated_by": &actor.UserID,
		"status":     "inactive",
	}).Error; err != nil {
		return apperror.Internal("Gagal menghapus pengguna", err)
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" && s.s3Client != nil {
		_ = s.s3Client.DeleteObjectByURL(ctx, *user.AvatarURL)
	}

	return nil
}

func (s *service) buildProfileResponse(user *User, role *Role, dept *Department, comp *Company, parentName *string, googleBound bool) *UserProfileResponse {
	resp := &UserProfileResponse{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		Image:       user.AvatarURL,
		Status:      user.Status,
		RoleID:      user.RoleID,
		RoleName:    role.Name,
		Role:        &RoleDTO{ID: role.ID, Name: role.Name, Color: role.Color},
		CompanyID:   user.CompanyID,
		ParentID:    user.ParentID,
		ParentName:  parentName,
		GoogleBound: googleBound,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	if dept != nil {
		resp.DepartmentID = &dept.ID
		resp.DepartmentName = &dept.Name
		resp.Department = &DepartmentDTO{
			ID:    dept.ID,
			Name:  dept.Name,
			Code:  dept.Code,
			Color: dept.Color,
		}
	}

	if comp != nil {
		resp.CompanyName = &comp.Name
		resp.Company = &CompanyDTO{
			ID:   comp.ID,
			Name: comp.Name,
			Slug: comp.Slug,
		}
	}

	return resp
}
