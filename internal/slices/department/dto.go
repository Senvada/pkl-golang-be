package department

import (
	"time"

	"github.com/google/uuid"
)

type CreateDepartmentRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Code  string `json:"code" validate:"required,min=2,max=20"`
	Color string `json:"color" validate:"omitempty,startswith=#,len=7"`
}

type UpdateDepartmentRequest struct {
	Name  *string `json:"name" validate:"omitempty,min=2,max=100"`
	Code  *string `json:"code" validate:"omitempty,min=2,max=20"`
	Color *string `json:"color" validate:"omitempty,startswith=#,len=7"`
}

type DepartmentResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Color     string    `json:"color"`
	UserCount int64     `json:"user_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
