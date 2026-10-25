package auth

import "github.com/google/uuid"

type RegisterRequest struct {
	Name         string     `json:"name" validate:"required,min=2,max=255"`
	Email        string     `json:"email" validate:"required,email"`
	Password     string     `json:"password" validate:"required,min=6"`
	RoleID       uuid.UUID  `json:"role_id" validate:"required"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	CompanyID    *uuid.UUID `json:"company_id,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"`
}

type GoogleAuthRequest struct {
	Credential string `json:"credential" validate:"required"`
}

type UserProfileDTO struct {
	ID            uuid.UUID                  `json:"id"`
	Name          string                     `json:"name"`
	Email         string                     `json:"email"`
	AvatarURL     *string                    `json:"avatar_url"`
	Status        string                     `json:"status"`
	RoleID        uuid.UUID                  `json:"role_id"`
	RoleName      string                     `json:"role_name"`
	DepartmentID  *uuid.UUID                 `json:"department_id,omitempty"`
	DepartmentIDs []uuid.UUID                `json:"department_ids,omitempty"`
	CompanyID     *uuid.UUID                 `json:"company_id,omitempty"`
	GoogleBound   bool                       `json:"google_bound"`
	Permissions   map[string]map[string]bool `json:"permissions,omitempty"`
}

type AuthResponse struct {
	User        UserProfileDTO `json:"user"`
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresIn   int64          `json:"expires_in"` // in seconds
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}
