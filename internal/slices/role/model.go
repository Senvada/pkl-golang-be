package role

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string         `gorm:"size:50;not null"`
	Color       string         `gorm:"size:7;not null;default:'#3B82F6'"`
	Permissions datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	RoleID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	Status    string         `gorm:"size:20;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
