package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoleID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	DepartmentID *uuid.UUID     `gorm:"type:uuid;index"`
	CompanyID    *uuid.UUID     `gorm:"type:uuid;index"`
	Name         string         `gorm:"size:255;not null"`
	Email        string         `gorm:"size:255;not null"`
	Password     string         `gorm:"type:text;not null"`
	AvatarURL    *string        `gorm:"type:text"`
	Status       string         `gorm:"size:20;not null;default:'active'"`
	CreatedBy    *uuid.UUID     `gorm:"type:uuid"`
	UpdatedBy    *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string         `gorm:"size:50;not null"`
	Color       string         `gorm:"size:7;not null;default:'#3B82F6'"`
	Permissions datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type Token struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"type:char(64);not null;index"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	Remember  bool       `gorm:"not null;default:false"`
	UserAgent *string    `gorm:"type:text"`
	IPAddress *string    `gorm:"size:45"`
	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	RevokedAt *time.Time
}

type UserGoogleAccount struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	GoogleID        string    `gorm:"size:255;not null;uniqueIndex"`
	GoogleEmail     string    `gorm:"size:255;not null"`
	GoogleName      *string   `gorm:"size:255"`
	GoogleAvatarURL *string   `gorm:"type:text"`
	EmailVerified   bool      `gorm:"not null;default:false"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
