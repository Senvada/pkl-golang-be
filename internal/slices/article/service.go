package article

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/authutil"
	"pkl-golang-be/internal/shared/pagination"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	GetArticles(ctx context.Context, actor *middleware.AuthClaims, filter ArticleFilterQuery, p pagination.Params) ([]ArticleResponse, pagination.Meta, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ArticleResponse, error)
	GetBySlug(ctx context.Context, slug string) (*ArticleResponse, error)
	Create(ctx context.Context, actor *middleware.AuthClaims, req CreateArticleRequest) (*ArticleResponse, error)
	Update(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateArticleRequest) (*ArticleResponse, error)
	UpdateStatus(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateArticleStatusRequest) (*ArticleResponse, error)
	Delete(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error
	GenerateCoverPresignedURL(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req PresignCoverRequest) (*PresignCoverResponse, error)
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

func generateSlug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonAlphaNumRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "article-" + uuid.New().String()[:8]
	}
	return s
}

func (s *service) GetArticles(ctx context.Context, actor *middleware.AuthClaims, filter ArticleFilterQuery, p pagination.Params) ([]ArticleResponse, pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&Article{}).
		Preload("Company").
		Preload("Author").
		Preload("Author.Role").
		Where("articles.deleted_at IS NULL")

	if filter.Search != "" {
		pat := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("(LOWER(articles.title) LIKE ?)", pat)
	}

	if filter.Status != "" {
		query = query.Where("articles.status = ?", filter.Status)
	}

	if filter.CompanyID != nil && *filter.CompanyID != uuid.Nil {
		query = query.Where("articles.company_id = ?", *filter.CompanyID)
	}

	if filter.UserID != nil && *filter.UserID != uuid.Nil {
		query = query.Where("articles.user_id = ?", *filter.UserID)
	}

	// Role and tenancy scoping
	isAdmin := actor != nil && strings.EqualFold(actor.RoleName, "admin")
	if actor == nil {
		// Public anonymous view: only published articles
		query = query.Where("articles.status = 'published'")
	} else if !isAdmin {
		if strings.EqualFold(actor.RoleName, "siswa") {
			// Siswa base role: can see their own articles (any status) or published articles
			query = query.Where("(articles.user_id = ? OR articles.status = 'published')", actor.UserID)
		} else if strings.EqualFold(actor.RoleName, "industri") && actor.CompanyID != nil {
			query = query.Where("(articles.company_id = ? OR articles.status = 'published')", *actor.CompanyID)
		} else {
			// Pembimbing / Guru / other roles with permissions and department bindings
			hasPublish, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "publish")
			hasUpdate, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "update")
			actorDeptIDs, _ := authutil.GetUserDepartmentIDs(ctx, s.db, actor.UserID)

			if len(actorDeptIDs) > 0 {
				deptUsersSubquery := s.db.Table("users").Select("id").Where("department_id IN (?)", actorDeptIDs)
				deptUsersJoinSubquery := s.db.Table("users_departments").Select("user_id").Where("department_id IN (?) AND deleted_at IS NULL", actorDeptIDs)
				deptCompaniesSubquery := s.db.Table("companies_departments").Select("company_id").Where("department_id IN (?) AND deleted_at IS NULL", actorDeptIDs)

				if hasUpdate {
					// Can update any article in their department scope
					query = query.Where("(articles.user_id = ? OR articles.status = 'published' OR articles.user_id IN (?) OR articles.user_id IN (?) OR articles.company_id IN (?))",
						actor.UserID, deptUsersSubquery, deptUsersJoinSubquery, deptCompaniesSubquery)
				} else if hasPublish {
					// Can see review articles in their department scope, plus published and own
					query = query.Where("(articles.user_id = ? OR articles.status = 'published' OR (articles.status = 'review' AND (articles.user_id IN (?) OR articles.user_id IN (?) OR articles.company_id IN (?))))",
						actor.UserID, deptUsersSubquery, deptUsersJoinSubquery, deptCompaniesSubquery)
				} else {
					// Write only (own articles) + published
					query = query.Where("(articles.user_id = ? OR articles.status = 'published')", actor.UserID)
				}
			} else {
				if hasUpdate {
					// No department restriction, can view all
				} else if hasPublish {
					query = query.Where("(articles.user_id = ? OR articles.status = 'published' OR articles.status = 'review')", actor.UserID)
				} else {
					query = query.Where("(articles.user_id = ? OR articles.status = 'published')", actor.UserID)
				}
			}
		}
	}

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, pagination.Meta{}, apperror.Internal("Gagal menghitung jumlah data artikel", err)
	}

	meta := p.BuildMeta(totalRows)

	var articles []Article
	if err := query.Scopes(p.Scope("created_at", "title", "status")).Find(&articles).Error; err != nil {
		return nil, pagination.Meta{}, apperror.Internal("Gagal memuat data artikel", err)
	}

	responses := make([]ArticleResponse, len(articles))
	for i, a := range articles {
		responses[i] = toArticleResponse(&a)
	}

	return responses, meta, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*ArticleResponse, error) {
	var a Article
	if err := s.db.WithContext(ctx).
		Preload("Company").
		Preload("Author").
		Preload("Author.Role").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Artikel tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal memuat data artikel", err)
	}

	res := toArticleResponse(&a)
	return &res, nil
}

func (s *service) GetBySlug(ctx context.Context, slug string) (*ArticleResponse, error) {
	var a Article
	if err := s.db.WithContext(ctx).
		Preload("Company").
		Preload("Author").
		Preload("Author.Role").
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Artikel tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal memuat data artikel", err)
	}

	res := toArticleResponse(&a)
	return &res, nil
}

func (s *service) Create(ctx context.Context, actor *middleware.AuthClaims, req CreateArticleRequest) (*ArticleResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("Autentikasi diperlukan")
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isSiswa := strings.EqualFold(actor.RoleName, "siswa")

	if !isAdmin && !isSiswa {
		hasWrite, err := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "write")
		if err != nil || !hasWrite {
			return nil, apperror.Forbidden("Anda tidak memiliki izin untuk membuat artikel baru")
		}
	}

	if isSiswa {
		var userCompanyID *uuid.UUID
		if actor.CompanyID != nil && *actor.CompanyID != uuid.Nil {
			userCompanyID = actor.CompanyID
		} else {
			var u struct {
				CompanyID *uuid.UUID `gorm:"column:company_id"`
			}
			if err := s.db.WithContext(ctx).Table("users").Select("company_id").Where("id = ? AND deleted_at IS NULL", actor.UserID).Scan(&u).Error; err == nil {
				userCompanyID = u.CompanyID
			}
		}

		if userCompanyID == nil || *userCompanyID == uuid.Nil {
			return nil, apperror.Forbidden("Akun siswa belum terikat dengan mitra industri. Anda belum dapat membuat artikel hingga terhubung dengan mitra industri.")
		}

		// Enforce that the student can only create articles for their bound company
		req.CompanyID = *userCompanyID
	}

	// Verify company exists
	var comp Company
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.CompanyID).First(&comp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("Perusahaan mitra tidak ditemukan", nil)
		}
		return nil, apperror.Internal("Gagal memverifikasi perusahaan mitra", err)
	}

	// Industri role can only create for their own company
	if strings.EqualFold(actor.RoleName, "industri") && actor.CompanyID != nil && *actor.CompanyID != req.CompanyID {
		return nil, apperror.Forbidden("Akun industri hanya dapat membuat artikel untuk perusahaannya sendiri")
	}

	baseSlug := generateSlug(req.Title)
	slug := baseSlug
	for i := 1; ; i++ {
		var count int64
		s.db.WithContext(ctx).Model(&Article{}).Where("company_id = ? AND slug = ? AND deleted_at IS NULL", req.CompanyID, slug).Count(&count)
		if count == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	status := "draft"
	if req.Status != "" {
		status = req.Status
	}

	if status == "published" {
		hasPublish, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "publish")
		if !hasPublish && !isAdmin {
			status = "draft" // Cannot directly publish without publish permission
		}
	}

	var content datatypes.JSON
	if req.Content != nil && len(*req.Content) > 0 {
		content = *req.Content
	} else {
		// Default to empty array []
		content = datatypes.JSON([]byte("[]"))
	}

	var style datatypes.JSON
	if req.Style != nil && len(*req.Style) > 0 {
		style = *req.Style
	} else {
		// Default to Style 1 (Klasik) height 400
		style = datatypes.JSON([]byte(`{"type": 1, "height": 400}`))
	}

	var publishedAt *time.Time
	if status == "published" {
		now := time.Now()
		publishedAt = &now
	}

	article := Article{
		CompanyID:   req.CompanyID,
		UserID:      actor.UserID,
		Title:       strings.TrimSpace(req.Title),
		Slug:        slug,
		Content:     content,
		Style:       style,
		CoverURL:    req.CoverURL,
		Status:      status,
		PublishedAt: publishedAt,
		CreatedBy:   &actor.UserID,
		UpdatedBy:   &actor.UserID,
	}

	if err := s.db.WithContext(ctx).Create(&article).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan artikel baru", err)
	}

	return s.GetByID(ctx, article.ID)
}

func (s *service) Update(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateArticleRequest) (*ArticleResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("Autentikasi diperlukan")
	}

	var existing Article
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Artikel tidak ditemukan")
		}
		return nil, apperror.Internal("Gagal memuat artikel", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isAuthor := existing.UserID == actor.UserID
	isCompanyOwner := strings.EqualFold(actor.RoleName, "industri") && actor.CompanyID != nil && *actor.CompanyID == existing.CompanyID
	isSiswa := strings.EqualFold(actor.RoleName, "siswa")

	hasUpdate, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "update")
	hasPublish, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "publish")
	hasWrite, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "write")

	// Check if this request is purely a status transition
	isStatusOnly := req.Status != nil && req.Title == nil && req.CompanyID == nil && req.CoverURL == nil && req.Style == nil && req.Content == nil

	if !isAdmin && !isCompanyOwner {
		if isAuthor {
			if !hasWrite && !isSiswa {
				return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengubah artikel ini")
			}
		} else {
			// Non-author:
			if isStatusOnly && hasPublish {
				// User with publish permission can transition status (e.g. review -> published or reject)
			} else if !hasUpdate {
				return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengubah artikel milik pengguna lain")
			}
		}
	}

	updates := make(map[string]any)

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title != "" && title != existing.Title {
			updates["title"] = title
			baseSlug := generateSlug(title)
			slug := baseSlug
			for i := 1; ; i++ {
				var count int64
				s.db.WithContext(ctx).Model(&Article{}).
					Where("company_id = ? AND slug = ? AND id != ? AND deleted_at IS NULL", existing.CompanyID, slug, id).
					Count(&count)
				if count == 0 {
					break
				}
				slug = fmt.Sprintf("%s-%d", baseSlug, i)
			}
			updates["slug"] = slug
		}
	}

	if req.CompanyID != nil && *req.CompanyID != existing.CompanyID {
		if isSiswa {
			return nil, apperror.Forbidden("Siswa tidak dapat mengubah perusahaan mitra yang telah ditentukan")
		}
		updates["company_id"] = *req.CompanyID
	}

	var oldCoverToDelete string
	if req.CoverURL != nil {
		newCover := strings.TrimSpace(*req.CoverURL)
		if existing.CoverURL != nil && *existing.CoverURL != "" && *existing.CoverURL != newCover {
			oldCoverToDelete = *existing.CoverURL
		}
		if newCover == "" {
			updates["cover_url"] = nil
		} else {
			updates["cover_url"] = newCover
		}
	}

	if req.Style != nil {
		updates["style"] = *req.Style
	}

	if req.Content != nil {
		updates["content"] = *req.Content
	}

	if req.Status != nil && *req.Status != "" {
		newStatus := *req.Status
		if newStatus == "published" {
			if !hasPublish && !isAdmin {
				return nil, apperror.Forbidden("Hanya pengguna dengan izin publikasi yang dapat merilis artikel ke publik")
			}
		}
		updates["status"] = newStatus
		if newStatus == "published" && existing.PublishedAt == nil {
			now := time.Now()
			updates["published_at"] = &now
		}
	}

	updates["updated_by"] = actor.UserID
	updates["updated_at"] = time.Now()

	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, apperror.Internal("Gagal menyimpan pembaruan artikel", err)
	}

	if oldCoverToDelete != "" && s.s3Client != nil {
		_ = s.s3Client.DeleteObjectByURL(ctx, oldCoverToDelete)
	}

	return s.GetByID(ctx, id)
}

func (s *service) UpdateStatus(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req UpdateArticleStatusRequest) (*ArticleResponse, error) {
	return s.Update(ctx, actor, id, UpdateArticleRequest{Status: &req.Status})
}

func (s *service) Delete(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID) error {
	if actor == nil {
		return apperror.Unauthorized("Autentikasi diperlukan")
	}

	var existing Article
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("Artikel tidak ditemukan")
		}
		return apperror.Internal("Gagal memuat artikel", err)
	}

	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isAuthor := existing.UserID == actor.UserID
	isCompanyOwner := strings.EqualFold(actor.RoleName, "industri") && actor.CompanyID != nil && *actor.CompanyID == existing.CompanyID
	hasDelete, _ := authutil.CheckRolePermission(ctx, s.db, actor.RoleID, actor.RoleName, "article", "delete")

	if !isAdmin && !isAuthor && !isCompanyOwner && !hasDelete {
		return apperror.Forbidden("Anda tidak memiliki izin untuk menghapus artikel ini")
	}

	if err := s.db.WithContext(ctx).Delete(&existing).Error; err != nil {
		return apperror.Internal("Gagal menghapus artikel", err)
	}

	if existing.CoverURL != nil && *existing.CoverURL != "" && s.s3Client != nil {
		_ = s.s3Client.DeleteObjectByURL(ctx, *existing.CoverURL)
	}

	return nil
}

func (s *service) GenerateCoverPresignedURL(ctx context.Context, actor *middleware.AuthClaims, id uuid.UUID, req PresignCoverRequest) (*PresignCoverResponse, error) {
	if s.s3Client == nil {
		return nil, apperror.Internal("Layanan penyimpanan cloud belum dikonfigurasi", nil)
	}

	folderID := "staging"
	if id != uuid.Nil {
		folderID = id.String()
	}

	ext := ".webp"
	switch strings.ToLower(req.ContentType) {
	case "image/jpeg", "image/jpg":
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

	key := fmt.Sprintf("articles/%s/cover_%d_%s%s", folderID, time.Now().Unix(), uuid.New().String()[:6], ext)

	uploadURL, err := s.s3Client.PresignPutObject(ctx, key, req.ContentType, 0, 10*time.Minute)
	if err != nil {
		return nil, apperror.Internal("Gagal membuat presigned upload URL", err)
	}

	publicURL := s.s3Client.GetPublicURL(key)

	return &PresignCoverResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
		Key:       key,
	}, nil
}

func toArticleResponse(a *Article) ArticleResponse {
	var comp CompanyBriefResponse
	if a.Company.ID != uuid.Nil {
		comp = CompanyBriefResponse{
			ID:       a.Company.ID,
			Name:     a.Company.Name,
			Slug:     a.Company.Slug,
			ImageURL: a.Company.ImageURL,
		}
	}

	var auth AuthorBriefResponse
	if a.Author.ID != uuid.Nil {
		auth = AuthorBriefResponse{
			ID:        a.Author.ID,
			Name:      a.Author.Name,
			Email:     a.Author.Email,
			AvatarURL: a.Author.AvatarURL,
			RoleName:  a.Author.Role.Name,
		}
	}

	return ArticleResponse{
		ID:          a.ID,
		CompanyID:   a.CompanyID,
		UserID:      a.UserID,
		Title:       a.Title,
		Slug:        a.Slug,
		Content:     a.Content,
		Style:       a.Style,
		CoverURL:    a.CoverURL,
		Status:      a.Status,
		PublishedAt: a.PublishedAt,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		Company:     comp,
		Author:      auth,
	}
}
