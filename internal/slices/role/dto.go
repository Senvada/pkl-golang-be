package role

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type RoleResponse struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Color       string         `json:"color"`
	Permissions datatypes.JSON `json:"permissions"`
	UserCount   int64          `json:"user_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateRoleRequest struct {
	Name        string         `json:"name" validate:"required,min=2,max=50"`
	Color       string         `json:"color" validate:"required,hexcolor"`
	Permissions datatypes.JSON `json:"permissions" validate:"required"`
}

type UpdateRoleRequest struct {
	Name        *string        `json:"name" validate:"omitempty,min=2,max=50"`
	Color       *string        `json:"color" validate:"omitempty,hexcolor"`
	Permissions datatypes.JSON `json:"permissions" validate:"omitempty"`
}
