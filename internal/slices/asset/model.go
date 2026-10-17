package asset

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Asset struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Path      string    `gorm:"type:text;not null"`
	Filename  string    `gorm:"size:255"`
	MimeType  string    `gorm:"size:100"`
	Size      int64     `gorm:"type:bigint"`
	CreatedBy uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
