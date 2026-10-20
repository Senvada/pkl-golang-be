package article

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type CreateArticleRequest struct {
	Title     string          `json:"title" validate:"required,min=3,max=255"`
	CompanyID uuid.UUID       `json:"company_id" validate:"required"`
	CoverURL  *string         `json:"cover_url,omitempty"`
	Style     *datatypes.JSON `json:"style,omitempty"`
	Content   *datatypes.JSON `json:"content,omitempty"`
	Status    string          `json:"status,omitempty" validate:"omitempty,oneof=draft review reject published archived"`
}

type UpdateArticleRequest struct {
	Title     *string         `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
	CompanyID *uuid.UUID      `json:"company_id,omitempty"`
	CoverURL  *string         `json:"cover_url,omitempty"`
	Style     *datatypes.JSON `json:"style,omitempty"`
	Content   *datatypes.JSON `json:"content,omitempty"`
	Status    *string         `json:"status,omitempty" validate:"omitempty,oneof=draft review reject published archived"`
}

type UpdateArticleStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft review reject published archived"`
}

type ArticleFilterQuery struct {
	Search    string     `query:"search"`
	Status    string     `query:"status"`
	CompanyID *uuid.UUID `query:"company_id"`
	UserID    *uuid.UUID `query:"user_id"`
}

type CompanyBriefResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	ImageURL *string   `json:"image_url,omitempty"`
}

type AuthorBriefResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	RoleName  string    `json:"role_name"`
}

type ArticleResponse struct {
	ID          uuid.UUID            `json:"id"`
	CompanyID   uuid.UUID            `json:"company_id"`
	UserID      uuid.UUID            `json:"user_id"`
	Title       string               `json:"title"`
	Slug        string               `json:"slug"`
	Content     datatypes.JSON       `json:"content"`
	Style       datatypes.JSON       `json:"style"`
	CoverURL    *string              `json:"cover_url,omitempty"`
	Status      string               `json:"status"`
	PublishedAt *time.Time           `json:"published_at,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	Company     CompanyBriefResponse `json:"company"`
	Author      AuthorBriefResponse  `json:"author"`
}

type PresignCoverRequest struct {
	ContentType string `json:"content_type" validate:"required"`
	FileSize    int64  `json:"file_size" validate:"required,max=10485760"`
}

type PresignCoverResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Key       string `json:"key"`
}
