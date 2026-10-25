package user

import (
	"time"

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
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color string    `json:"color"`
}

type DepartmentDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Code  string    `json:"code"`
	Color string    `json:"color"`
}

type CompanyDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type UserProfileResponse struct {
	ID             uuid.UUID                  `json:"id"`
	Name           string                     `json:"name"`
	Email          string                     `json:"email"`
	AvatarURL      *string                    `json:"avatar_url"`
	Image          *string                    `json:"image"` // Alias for frontend backward compatibility
	Status         string                     `json:"status"`
	RoleID         uuid.UUID                  `json:"role_id"`
	RoleName       string                     `json:"role_name"`
	Role           *RoleDTO                   `json:"role,omitempty"`
	DepartmentID   *uuid.UUID                 `json:"department_id,omitempty"`
	DepartmentName *string                    `json:"department_name,omitempty"`
	Department     *DepartmentDTO             `json:"department,omitempty"`
	DepartmentIDs  []uuid.UUID                `json:"department_ids,omitempty"`
	Departments    []DepartmentDTO            `json:"departments,omitempty"`
	CompanyID      *uuid.UUID                 `json:"company_id,omitempty"`
	CompanyName    *string                    `json:"company_name,omitempty"`
	Company        *CompanyDTO                `json:"company,omitempty"`
	ParentID       *uuid.UUID                 `json:"parent_id,omitempty"`
	ParentName     *string                    `json:"parent_name,omitempty"`
	GoogleBound    bool                       `json:"google_bound"`
	Permissions    map[string]map[string]bool `json:"permissions,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type CreateUserRequest struct {
	Name          string      `json:"name" validate:"required,min=2,max=255"`
	Email         string      `json:"email" validate:"required,email"`
	Password      string      `json:"password" validate:"required,min=6"`
	RoleID        uuid.UUID   `json:"role_id" validate:"required"`
	DepartmentID  *uuid.UUID  `json:"department_id" validate:"omitempty"`
	DepartmentIDs []uuid.UUID `json:"department_ids" validate:"omitempty"`
	CompanyID     *uuid.UUID  `json:"company_id" validate:"omitempty"`
	ParentID      *uuid.UUID  `json:"parent_id" validate:"omitempty"`
	Status        string      `json:"status" validate:"omitempty,oneof=active waiting inactive banned"`
}

type UpdateUserRequest struct {
	Name          *string      `json:"name" validate:"omitempty,min=2,max=255"`
	Email         *string      `json:"email" validate:"omitempty,email"`
	Password      *string      `json:"password" validate:"omitempty,min=6"`
	RoleID        *uuid.UUID   `json:"role_id" validate:"omitempty"`
	DepartmentID  *uuid.UUID   `json:"department_id" validate:"omitempty"`
	DepartmentIDs *[]uuid.UUID `json:"department_ids" validate:"omitempty"`
	CompanyID     *uuid.UUID   `json:"company_id" validate:"omitempty"`
	ParentID      *uuid.UUID   `json:"parent_id" validate:"omitempty"`
	Status        *string      `json:"status" validate:"omitempty,oneof=active waiting inactive banned"`
	AvatarURL     *string      `json:"avatar_url" validate:"omitempty"`
}

type UserFilterQuery struct {
	Search       string     `query:"search"`
	RoleID       *uuid.UUID `query:"role_id"`
	DepartmentID *uuid.UUID `query:"department_id"`
	CompanyID    *uuid.UUID `query:"company_id"`
	ParentID     *uuid.UUID `query:"parent_id"`
	Status       string     `query:"status"`
}

type UserItemResponse struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	Email          string          `json:"email"`
	AvatarURL      *string         `json:"avatar_url"`
	Image          *string         `json:"image"`
	Status         string          `json:"status"`
	RoleID         uuid.UUID       `json:"role_id"`
	RoleName       string          `json:"role_name"`
	Role           *RoleDTO        `json:"role,omitempty"`
	DepartmentID   *uuid.UUID      `json:"department_id,omitempty"`
	DepartmentName *string         `json:"department_name,omitempty"`
	Department     *DepartmentDTO  `json:"department,omitempty"`
	DepartmentIDs  []uuid.UUID     `json:"department_ids,omitempty"`
	Departments    []DepartmentDTO `json:"departments,omitempty"`
	CompanyID      *uuid.UUID      `json:"company_id,omitempty"`
	CompanyName    *string         `json:"company_name,omitempty"`
	Company        *CompanyDTO     `json:"company,omitempty"`
	ParentID       *uuid.UUID      `json:"parent_id,omitempty"`
	ParentName     *string         `json:"parent_name,omitempty"`
	SubUserCount   int64           `json:"sub_user_count"`
	GoogleBound    bool            `json:"google_bound"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
