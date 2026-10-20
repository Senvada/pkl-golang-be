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

	// Role scoping
	isAdmin := actor != nil && strings.EqualFold(actor.RoleName, "admin")
	if !isAdmin && actor != nil {
		if strings.EqualFold(actor.RoleName, "industri") && actor.CompanyID != nil {
			query = query.Where("articles.company_id = ?", *actor.CompanyID)
		} else if strings.EqualFold(actor.RoleName, "siswa") {
			// Siswa sees their own articles, or published articles
			query = query.Where("(articles.user_id = ? OR articles.status = 'published')", actor.UserID)
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

	// Permission check: Admin, or Author, or Industri manager
	isAdmin := strings.EqualFold(actor.RoleName, "admin")
	isAuthor := existing.UserID == actor.UserID
	isCompanyOwner := strings.EqualFold(actor.RoleName, "industri") && actor.CompanyID != nil && *actor.CompanyID == existing.CompanyID

	if !isAdmin && !isAuthor && !isCompanyOwner {
		return nil, apperror.Forbidden("Anda tidak memiliki izin untuk mengubah artikel ini")
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

	if !isAdmin && !isAuthor && !isCompanyOwner {
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
