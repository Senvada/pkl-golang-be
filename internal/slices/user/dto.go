package user

import (
	"github.com/google/uuid"
)

type PresignAvatarRequest struct {
	ContentType string `json:"content_type" validate:"required,oneof=image/jpeg image/png image/webp image/gif"`
	FileSize    int64  `json:"file_size" validate:"required,gt=0,lte=5242880"` // Max 5 MB
}

type PresignAvatarResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Key       string `json:"key"`
}

type UpdateProfileRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=255"`
	AvatarURL   *string `json:"avatar_url" validate:"omitempty"`
	PasswordOld *string `json:"passwordOld" validate:"omitempty"`
	PasswordNew *string `json:"passwordNew" validate:"omitempty,min=6"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

type RoleDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type DepartmentDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color string    `json:"color"`
}

type UserProfileResponse struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	Email        string         `json:"email"`
	AvatarURL    *string        `json:"avatar_url"`
	Image        *string        `json:"image"` // Alias for frontend backward compatibility
	Status       string         `json:"status"`
	RoleID       uuid.UUID      `json:"role_id"`
	RoleName     string         `json:"role_name"`
	Role         *RoleDTO       `json:"role,omitempty"`
	DepartmentID *uuid.UUID     `json:"department_id,omitempty"`
	Department   *DepartmentDTO `json:"department,omitempty"`
	CompanyID    *uuid.UUID     `json:"company_id,omitempty"`
	GoogleBound  bool           `json:"google_bound"`
}
