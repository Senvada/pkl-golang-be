package article

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Article struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CompanyID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index"`
	Title       string         `gorm:"size:255;not null"`
	Slug        string         `gorm:"size:255;not null"`
	Content     datatypes.JSON `gorm:"type:jsonb"`
	Style       datatypes.JSON `gorm:"type:jsonb"`
	CoverURL    *string        `gorm:"type:text"`
	Status      string         `gorm:"size:20;not null;default:'draft'"`
	PublishedAt *time.Time
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Company Company `gorm:"foreignKey:CompanyID;references:ID"`
	Author  User    `gorm:"foreignKey:UserID;references:ID"`
}

type Company struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name     string    `gorm:"size:255;not null"`
	Slug     string    `gorm:"size:255;not null"`
	ImageURL *string   `gorm:"type:text"`
}

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:100;not null"`
	Email     string    `gorm:"size:255;not null"`
	AvatarURL *string   `gorm:"type:text"`
	RoleID    uuid.UUID `gorm:"type:uuid;not null"`
	Role      Role      `gorm:"foreignKey:RoleID;references:ID"`
}

type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name string    `gorm:"size:50;not null"`
}

func (Article) TableName() string {
	return "articles"
}

func (Company) TableName() string {
	return "companies"
}

func (User) TableName() string {
	return "users"
}

func (Role) TableName() string {
	return "roles"
}
