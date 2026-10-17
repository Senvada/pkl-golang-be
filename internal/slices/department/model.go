package department

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Department struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"size:100;not null"`
	Code      string    `gorm:"size:20;not null"`
	Color     string    `gorm:"size:7;not null;default:'#10B981'"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	DepartmentID *uuid.UUID     `gorm:"type:uuid;index"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
