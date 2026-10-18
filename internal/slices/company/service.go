package company

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/pagination"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	GetCompanies(ctx context.Context, filter CompanyFilterQuery, p pagination.Params) ([]CompanyResponse, pagination.Meta, error)
	GetAll(ctx context.Context) ([]CompanyResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CompanyResponse, error)
	GetBySlug(ctx context.Context, slug string) (*CompanyResponse, error)
	GetMaps(ctx context.Context) ([]CompanyMapResponse, error)
	Create(ctx context.Context, actor *middleware.AuthClaims, req CreateCompanyRequest) (*CompanyResponse, error)
	Update(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateCompanyRequest) (*CompanyResponse, error)
	UpdateContent(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, content datatypes.JSON) (*CompanyResponse, error)
	Delete(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error
	GenerateImagePresignedURL(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req PresignImageRequest) (*PresignImageResponse, error)
	ResolveMaps(ctx context.Context, rawURL string) (*ResolveMapsResponse, error)
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

func (s *service) GetCompanies(ctx context.Context, filter CompanyFilterQuery, p pagination.Params) ([]CompanyResponse, pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&Company{}).Where("companies.deleted_at IS NULL")

	if filter.Search != "" {
		pat := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("(LOWER(companies.name) LIKE ? OR LOWER(companies.address) LIKE ? OR LOWER(companies.description) LIKE ?)", pat, pat, pat)
	}

	if filter.Status != "" {
		query = query.Where("companies.status = ?", filter.Status)
	}

	if filter.DepartmentID != nil {
		query = query.Where("companies.id IN (SELECT company_id FROM companies_departments WHERE department_id = ? AND deleted_at IS NULL)", *filter.DepartmentID)
	}

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, pagination.Meta{}, apperror.Internal("Gagal menghitung jumlah data perusahaan mitra", err)
	}

	meta := p.BuildMeta(totalRows)

	var companies []Company
	if err := query.Scopes(p.Scope("created_at", "name", "status")).Find(&companies).Error; err != nil {
		return nil, meta, apperror.Internal("Gagal mengambil data perusahaan mitra", err)
	}

	if len(companies) == 0 {
		return []CompanyResponse{}, meta, nil
	}

	responses, err := s.populateCompanyExtras(ctx, companies)
	if err != nil {
		return nil, meta, err
	}

	return responses, meta, nil
}

func (s *service) GetAll(ctx context.Context) ([]CompanyResponse, error) {
	var companies []Company
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Find(&companies).Error; err != nil {
		return nil, apperror.Internal("Gagal mengambil data perusahaan mitra", err)
	}

	if len(companies) == 0 {
		return []CompanyResponse{}, nil
	}

	return s.populateCompanyExtras(ctx, companies)
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan mitra tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil detail perusahaan mitra", err)
	}

	res, err := s.populateCompanyExtras(ctx, []Company{c})
	if err != nil || len(res) == 0 {
		return nil, err
	}

	return &res[0], nil
}

func (s *service) GetBySlug(ctx context.Context, slug string) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan mitra tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal mengambil detail perusahaan mitra", err)
	}

	res, err := s.populateCompanyExtras(ctx, []Company{c})
	if err != nil || len(res) == 0 {
		return nil, err
	}

	return &res[0], nil
}

func (s *service) GetMaps(ctx context.Context) ([]CompanyMapResponse, error) {
	var companies []Company
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND status = 'active' AND lat IS NOT NULL AND lng IS NOT NULL").
		Order("name ASC").
		Find(&companies).Error; err != nil {
		return nil, apperror.Internal("Gagal mengambil data peta mitra industri", err)
	}

	if len(companies) == 0 {
		return []CompanyMapResponse{}, nil
	}

	compIDs := make([]uuid.UUID, len(companies))
	for i, c := range companies {
		compIDs[i] = c.ID
	}

	type cdRow struct {
		CompanyID    uuid.UUID `gorm:"column:company_id"`
		DepartmentID uuid.UUID `gorm:"column:department_id"`
		Name         string    `gorm:"column:name"`
		Code         string    `gorm:"column:code"`
		Color        string    `gorm:"column:color"`
	}
	var cdRows []cdRow
	_ = s.db.WithContext(ctx).
		Table("companies_departments").
		Select("companies_departments.company_id, departments.id as department_id, departments.name, departments.code, departments.color").
		Joins("JOIN departments ON departments.id = companies_departments.department_id AND departments.deleted_at IS NULL").
		Where("companies_departments.company_id IN ? AND companies_departments.deleted_at IS NULL", compIDs).
		Scan(&cdRows)

	deptMap := make(map[uuid.UUID][]DepartmentDTO)
	for _, row := range cdRows {
		deptMap[row.CompanyID] = append(deptMap[row.CompanyID], DepartmentDTO{
			ID:    row.DepartmentID,
			Name:  row.Name,
			Code:  row.Code,
			Color: row.Color,
		})
	}

	results := make([]CompanyMapResponse, len(companies))
	for i, c := range companies {
		lat := 0.0
		if c.Lat != nil {
			lat = *c.Lat
		}
		lng := 0.0
		if c.Lng != nil {
			lng = *c.Lng
		}

		depts := deptMap[c.ID]
		if depts == nil {
			depts = []DepartmentDTO{}
		}

		results[i] = CompanyMapResponse{
			ID:          c.ID,
			Name:        c.Name,
			Slug:        c.Slug,
			Address:     c.Address,
			Description: c.Description,
			ImageURL:    c.ImageURL,
			GmapsLink:   c.GmapsLink,
			Lat:         lat,
			Lng:         lng,
			Departments: depts,
		}
	}

	return results, nil
}

func (s *service) Create(ctx context.Context, actor *middleware.AuthClaims, req CreateCompanyRequest) (*CompanyResponse, error) {
	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	if !isAdmin {
		return nil, apperror.Forbidden("Hanya administrator yang dapat mendaftarkan perusahaan mitra baru")
	}

	baseSlug := generateSlug(req.Name)
	slug := baseSlug

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

	company := Company{
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
		CreatedBy:   &actor.UserID,
		UpdatedBy:   &actor.UserID,
	}

	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Create(&company).Error; err != nil {
		tx.Rollback()
		return nil, apperror.Internal("Gagal menyimpan data perusahaan mitra", err)
	}

	// Insert accepted departments
	if len(req.DepartmentIDs) > 0 {
		for _, dID := range req.DepartmentIDs {
			cd := CompanyDepartment{
				CompanyID:    company.ID,
				DepartmentID: dID,
			}
			if err := tx.Create(&cd).Error; err != nil {
				tx.Rollback()
				return nil, apperror.Internal("Gagal menyimpan pemetaan jurusan perusahaan", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("Gagal menyelesaikan transaksi", err)
	}

	return s.GetByID(ctx, company.ID)
}

func (s *service) Update(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateCompanyRequest) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan mitra tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan perusahaan mitra", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isMyCompany := actor.CompanyID != nil && *actor.CompanyID == id

	if !isAdmin && !isMyCompany {
		return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengubah data perusahaan ini")
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
	// Status change is restricted to Admin only
	if req.Status != nil && isAdmin {
		updates["status"] = *req.Status
	}

	tx := s.db.WithContext(ctx).Begin()

	if len(updates) > 0 {
		updates["updated_by"] = &actor.UserID
		updates["updated_at"] = time.Now()
		if err := tx.Model(&c).Updates(updates).Error; err != nil {
			tx.Rollback()
			return nil, apperror.Internal("Gagal memperbarui data perusahaan", err)
		}
	}

	// Update department associations if provided (Admin only)
	if req.DepartmentIDs != nil && isAdmin {
		// Hard delete old mappings for this company
		if err := tx.Exec("DELETE FROM companies_departments WHERE company_id = ?", id).Error; err != nil {
			tx.Rollback()
			return nil, apperror.Internal("Gagal memperbarui pemetaan jurusan", err)
		}

		for _, dID := range *req.DepartmentIDs {
			cd := CompanyDepartment{
				CompanyID:    id,
				DepartmentID: dID,
			}
			if err := tx.Create(&cd).Error; err != nil {
				tx.Rollback()
				return nil, apperror.Internal("Gagal menambahkan pemetaan jurusan baru", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan perubahan", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) UpdateContent(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, content datatypes.JSON) (*CompanyResponse, error) {
	var c Company
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Perusahaan mitra tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal menemukan perusahaan", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isMyCompany := actor.CompanyID != nil && *actor.CompanyID == id

	if !isAdmin && !isMyCompany {
		return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengedit konten perusahaan ini")
	}

	if err := s.db.WithContext(ctx).Model(&c).Updates(map[string]any{
		"content":    content,
		"updated_by": &actor.UserID,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, apperror.Internal("Gagal memperbarui konten perusahaan", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error {
	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	if !isAdmin {
		return apperror.Forbidden("Hanya administrator yang dapat menghapus data perusahaan mitra")
	}

	var c Company
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Perusahaan mitra tidak ditemukan")
		}
		return apperror.Internal("Gagal menemukan perusahaan mitra", err)
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&c).Updates(map[string]any{
		"deleted_at": now,
		"updated_by": &actor.UserID,
		"status":     "inactive",
	}).Error; err != nil {
		return apperror.Internal("Gagal menghapus perusahaan mitra", err)
	}

	return nil
}

func (s *service) GenerateImagePresignedURL(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req PresignImageRequest) (*PresignImageResponse, error) {
	if s.s3Client == nil {
		return nil, apperror.Internal("Layanan penyimpanan storage belum diinisialisasi", errors.New("s3 client is nil"))
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isMyCompany := actor.CompanyID != nil && (id == uuid.Nil || *actor.CompanyID == id)

	if !isAdmin && !isMyCompany {
		return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengunggah gambar untuk perusahaan ini")
	}

	var folderID string
	if id != uuid.Nil {
		folderID = id.String()
	} else if actor.CompanyID != nil {
		folderID = actor.CompanyID.String()
	} else {
		folderID = "staging"
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

	key := fmt.Sprintf("companies/%s/%d_%s%s", folderID, time.Now().Unix(), uuid.New().String()[:8], ext)

	uploadURL, err := s.s3Client.PresignPutObject(ctx, key, req.ContentType, 0, 10*time.Minute)
	if err != nil {
		return nil, apperror.Internal("Gagal membuat presigned upload URL", err)
	}

	publicURL := s.s3Client.GetPublicURL(key)

	return &PresignImageResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
		Key:       key,
	}, nil
}

// -----------------------------------------------------------------------
// HELPER: Populate Departments & Member Counts
// -----------------------------------------------------------------------

func (s *service) populateCompanyExtras(ctx context.Context, companies []Company) ([]CompanyResponse, error) {
	compIDs := make([]uuid.UUID, len(companies))
	for i, c := range companies {
		compIDs[i] = c.ID
	}

	// 1. Member Counts (Users associated with company_id)
	type memberCountRow struct {
		CompanyID uuid.UUID `gorm:"column:company_id"`
		Count     int64     `gorm:"column:count"`
	}
	var memberCounts []memberCountRow
	_ = s.db.WithContext(ctx).
		Table("users").
		Select("company_id, count(*) as count").
		Where("company_id IN ? AND deleted_at IS NULL", compIDs).
		Group("company_id").
		Scan(&memberCounts)

	countMap := make(map[uuid.UUID]int64, len(memberCounts))
	for _, mc := range memberCounts {
		countMap[mc.CompanyID] = mc.Count
	}

	// 2. Department mappings
	type cdRow struct {
		CompanyID    uuid.UUID `gorm:"column:company_id"`
		DepartmentID uuid.UUID `gorm:"column:department_id"`
		Name         string    `gorm:"column:name"`
		Code         string    `gorm:"column:code"`
		Color        string    `gorm:"column:color"`
	}
	var cdRows []cdRow
	_ = s.db.WithContext(ctx).
		Table("companies_departments").
		Select("companies_departments.company_id, departments.id as department_id, departments.name, departments.code, departments.color").
		Joins("JOIN departments ON departments.id = companies_departments.department_id AND departments.deleted_at IS NULL").
		Where("companies_departments.company_id IN ? AND companies_departments.deleted_at IS NULL", compIDs).
		Scan(&cdRows)

	deptMap := make(map[uuid.UUID][]DepartmentDTO)
	for _, row := range cdRows {
		deptMap[row.CompanyID] = append(deptMap[row.CompanyID], DepartmentDTO{
			ID:    row.DepartmentID,
			Name:  row.Name,
			Code:  row.Code,
			Color: row.Color,
		})
	}

	responses := make([]CompanyResponse, len(companies))
	for i, c := range companies {
		depts := deptMap[c.ID]
		if depts == nil {
			depts = []DepartmentDTO{}
		}

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
			Departments: depts,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		}
	}

	return responses, nil
}

var (
	latLngAtRegex    = regexp.MustCompile(`@(-?\d+\.\d+),(-?\d+\.\d+)`)
	latLngQueryRegex = regexp.MustCompile(`[?&](?:q|ll|destination)=(-?\d+\.\d+),(-?\d+\.\d+)`)
	latLngProtoRegex = regexp.MustCompile(`!3d(-?\d+\.\d+)!4d(-?\d+\.\d+)`)
)

func extractLatLngFromURL(rawURL string) *ResolveMapsResponse {
	// 1. Check for @lat,lng
	if match := latLngAtRegex.FindStringSubmatch(rawURL); len(match) == 3 {
		lat, errLat := strconv.ParseFloat(match[1], 64)
		lng, errLng := strconv.ParseFloat(match[2], 64)
		if errLat == nil && errLng == nil {
			return &ResolveMapsResponse{Lat: lat, Lng: lng}
		}
	}

	// 2. Check for Protobuf embedded format !3dLat!4dLng
	if match := latLngProtoRegex.FindStringSubmatch(rawURL); len(match) == 3 {
		lat, errLat := strconv.ParseFloat(match[1], 64)
		lng, errLng := strconv.ParseFloat(match[2], 64)
		if errLat == nil && errLng == nil {
			return &ResolveMapsResponse{Lat: lat, Lng: lng}
		}
	}

	// 3. Check for Query parameters ?q=lat,lng or ?ll=lat,lng or ?destination=lat,lng
	if match := latLngQueryRegex.FindStringSubmatch(rawURL); len(match) == 3 {
		lat, errLat := strconv.ParseFloat(match[1], 64)
		lng, errLng := strconv.ParseFloat(match[2], 64)
		if errLat == nil && errLng == nil {
			return &ResolveMapsResponse{Lat: lat, Lng: lng}
		}
	}

	return nil
}

func (s *service) ResolveMaps(ctx context.Context, rawURL string) (*ResolveMapsResponse, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, apperror.BadRequest("URL Google Maps wajib diisi", nil)
	}

	// 1. Direct regex check on original URL (instant, no network request needed)
	if coords := extractLatLngFromURL(rawURL); coords != nil {
		return coords, nil
	}

	// 2. Resolve redirects for shortened links (e.g. maps.app.goo.gl or goo.gl/maps)
	var redirectURLs []string
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectURLs = append(redirectURLs, req.URL.String())
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, apperror.BadRequest("URL tidak valid", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		// Even if redirect error or connection ended, check all captured redirect URLs
		for _, u := range redirectURLs {
			if coords := extractLatLngFromURL(u); coords != nil {
				return coords, nil
			}
		}
		return nil, apperror.BadRequest("Gagal menghubungi tautan Google Maps", err)
	}
	defer resp.Body.Close()

	// Check final redirected URL
	if resp.Request != nil && resp.Request.URL != nil {
		if coords := extractLatLngFromURL(resp.Request.URL.String()); coords != nil {
			return coords, nil
		}
	}

	// Check all intermediate redirect URLs
	for _, u := range redirectURLs {
		if coords := extractLatLngFromURL(u); coords != nil {
			return coords, nil
		}
	}

	// Check Location header if present
	if loc := resp.Header.Get("Location"); loc != "" {
		if coords := extractLatLngFromURL(loc); coords != nil {
			return coords, nil
		}
	}

	return nil, apperror.NotFound("Tidak dapat menemukan koordinat dari tautan Google Maps tersebut. Pastikan tautan mengarah ke lokasi pin peta.")
}
