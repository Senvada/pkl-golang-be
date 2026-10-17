package company

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type CreateCompanyRequest struct {
	Name        string          `json:"name" validate:"required,min=2,max=255"`
	Email       *string         `json:"email" validate:"omitempty,email"`
	PhoneNumber *string         `json:"phone_number" validate:"omitempty,max=30"`
	Address     *string         `json:"address" validate:"omitempty"`
	Description *string         `json:"description" validate:"omitempty"`
	Content     *datatypes.JSON `json:"content" validate:"omitempty"`
	ImageURL    *string         `json:"image_url" validate:"omitempty"`
	GmapsLink   *string         `json:"gmaps_link" validate:"omitempty"`
	Lat         *float64        `json:"lat" validate:"omitempty"`
	Lng         *float64        `json:"lng" validate:"omitempty"`
	Status      string          `json:"status" validate:"omitempty,oneof=active inactive"`
}

type UpdateCompanyRequest struct {
	Name        *string         `json:"name" validate:"omitempty,min=2,max=255"`
	Email       *string         `json:"email" validate:"omitempty,email"`
	PhoneNumber *string         `json:"phone_number" validate:"omitempty,max=30"`
	Address     *string         `json:"address" validate:"omitempty"`
	Description *string         `json:"description" validate:"omitempty"`
	Content     *datatypes.JSON `json:"content" validate:"omitempty"`
	ImageURL    *string         `json:"image_url" validate:"omitempty"`
	GmapsLink   *string         `json:"gmaps_link" validate:"omitempty"`
	Lat         *float64        `json:"lat" validate:"omitempty"`
	Lng         *float64        `json:"lng" validate:"omitempty"`
	Status      *string         `json:"status" validate:"omitempty,oneof=active inactive"`
}

type CompanyResponse struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Email       *string        `json:"email"`
	PhoneNumber *string        `json:"phone_number"`
	Address     *string        `json:"address"`
	Description *string        `json:"description"`
	Content     datatypes.JSON `json:"content,omitempty"`
	ImageURL    *string        `json:"image_url"`
	GmapsLink   *string        `json:"gmaps_link"`
	Lat         *float64       `json:"lat"`
	Lng         *float64       `json:"lng"`
	Status      string         `json:"status"`
	MemberCount int64          `json:"member_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
