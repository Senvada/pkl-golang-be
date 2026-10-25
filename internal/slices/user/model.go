package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoleID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	DepartmentID *uuid.UUID `gorm:"type:uuid;index"`
	CompanyID    *uuid.UUID `gorm:"type:uuid;index"`
	ParentID     *uuid.UUID `gorm:"type:uuid;index"`
	Name         string     `gorm:"size:255;not null"`
	Email        string     `gorm:"size:255;not null"`
	Password     string     `gorm:"type:text;not null"`
	AvatarURL    *string    `gorm:"type:text"`
	Status       string     `gorm:"size:20;not null;default:'active'"`
	CreatedBy    *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy    *uuid.UUID `gorm:"type:uuid"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type Role struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:50;not null"`
	Color     string    `gorm:"size:7;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Department struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:100;not null"`
	Code      string    `gorm:"size:20;not null"`
	Color     string    `gorm:"size:7;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Company struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name      string         `gorm:"size:255;not null"`
	Slug      string         `gorm:"size:255;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type UserGoogleAccount struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	GoogleID string    `gorm:"size:255;not null"`
}

type UserDepartment struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index"`
	DepartmentID uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (UserDepartment) TableName() string {
	return "users_departments"
}
