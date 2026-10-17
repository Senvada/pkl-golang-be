package company

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Company struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string         `gorm:"size:255;not null"`
	Slug        string         `gorm:"size:255;not null"`
	Email       *string        `gorm:"size:255"`
	PhoneNumber *string        `gorm:"size:30"`
	Address     *string        `gorm:"type:text"`
	Description *string        `gorm:"type:text"`
	Content     datatypes.JSON `gorm:"type:jsonb"`
	ImageURL    *string        `gorm:"type:text"`
	GmapsLink   *string        `gorm:"type:text"`
	Lat         *float64       `gorm:"type:decimal(10,8)"`
	Lng         *float64       `gorm:"type:decimal(11,8)"`
	Status      string         `gorm:"size:20;not null;default:'active'"`
	CreatedBy   *uuid.UUID     `gorm:"type:uuid"`
	UpdatedBy   *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
