package company

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type DepartmentDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Code  string    `json:"code"`
	Color string    `json:"color"`
}

type PresignImageRequest struct {
	ContentType string `json:"content_type" validate:"required,oneof=image/jpeg image/png image/webp image/gif"`
	FileSize    int64  `json:"file_size" validate:"required,gt=0,lte=10485760"` // Max 10 MB
}

type PresignImageResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Key       string `json:"key"`
}

type CreateCompanyRequest struct {
	Name          string          `json:"name" validate:"required,min=2,max=255"`
	Email         *string         `json:"email" validate:"omitempty,email"`
	PhoneNumber   *string         `json:"phone_number" validate:"omitempty,max=30"`
	Address       *string         `json:"address" validate:"omitempty"`
	Description   *string         `json:"description" validate:"omitempty"`
	Content       *datatypes.JSON `json:"content" validate:"omitempty"`
	ImageURL      *string         `json:"image_url" validate:"omitempty"`
	GmapsLink     *string         `json:"gmaps_link" validate:"omitempty"`
	Lat           *float64        `json:"lat" validate:"omitempty"`
	Lng           *float64        `json:"lng" validate:"omitempty"`
	Status        string          `json:"status" validate:"omitempty,oneof=active inactive"`
	DepartmentIDs []uuid.UUID     `json:"department_ids" validate:"omitempty"`
}

type UpdateCompanyRequest struct {
	Name          *string         `json:"name" validate:"omitempty,min=2,max=255"`
	Email         *string         `json:"email" validate:"omitempty,email"`
	PhoneNumber   *string         `json:"phone_number" validate:"omitempty,max=30"`
	Address       *string         `json:"address" validate:"omitempty"`
	Description   *string         `json:"description" validate:"omitempty"`
	Content       *datatypes.JSON `json:"content" validate:"omitempty"`
	ImageURL      *string         `json:"image_url" validate:"omitempty"`
	GmapsLink     *string         `json:"gmaps_link" validate:"omitempty"`
	Lat           *float64        `json:"lat" validate:"omitempty"`
	Lng           *float64        `json:"lng" validate:"omitempty"`
	Status        *string         `json:"status" validate:"omitempty,oneof=active inactive"`
	DepartmentIDs *[]uuid.UUID    `json:"department_ids" validate:"omitempty"`
}

type UpdateCompanyContentRequest struct {
	Content datatypes.JSON `json:"content" validate:"required"`
}

type CompanyFilterQuery struct {
	Search       string     `query:"search"`
	DepartmentID *uuid.UUID `query:"department_id"`
	Status       string     `query:"status"`
}

type CompanyResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Email       *string         `json:"email"`
	PhoneNumber *string         `json:"phone_number"`
	Address     *string         `json:"address"`
	Description *string         `json:"description"`
	Content     datatypes.JSON  `json:"content,omitempty"`
	ImageURL    *string         `json:"image_url"`
	GmapsLink   *string         `json:"gmaps_link"`
	Lat         *float64        `json:"lat"`
	Lng         *float64        `json:"lng"`
	Status      string          `json:"status"`
	MemberCount int64           `json:"member_count"`
	Departments []DepartmentDTO `json:"departments"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CompanyMapResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Address     *string         `json:"address"`
	Description *string         `json:"description"`
	ImageURL    *string         `json:"image_url"`
	GmapsLink   *string         `json:"gmaps_link"`
	Lat         float64         `json:"lat"`
	Lng         float64         `json:"lng"`
	Departments []DepartmentDTO `json:"departments"`
}

type ResolveMapsRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type ResolveMapsResponse struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type CompanyDepartmentResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Code  string    `json:"code"`
	Color string    `json:"color"`
}

type CompanyInternSupervisorResponse struct {
	ID         uuid.UUID                  `json:"id"`
	Name       string                     `json:"name"`
	AvatarURL  *string                    `json:"avatar_url"`
	RoleName   string                     `json:"role_name"`
	Department *CompanyDepartmentResponse `json:"department,omitempty"`
}

type CompanyInternArticleResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	CoverURL    *string    `json:"cover_url"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CompanyInternResponse struct {
	ID         uuid.UUID                        `json:"id"`
	Name       string                           `json:"name"`
	AvatarURL  *string                          `json:"avatar_url"`
	Department *CompanyDepartmentResponse       `json:"department,omitempty"`
	Supervisor *CompanyInternSupervisorResponse `json:"supervisor,omitempty"`
	Articles   []CompanyInternArticleResponse   `json:"articles"`
}
