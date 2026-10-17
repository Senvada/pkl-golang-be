package asset

import (
	"time"

	"github.com/google/uuid"
)

type PresignAssetRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"content_type" validate:"required"`
	Size        int64  `json:"size" validate:"omitempty,min=0"`
	Folder      string `json:"folder" validate:"omitempty,max=50"`
}

type PresignAssetResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Path      string `json:"path"`
}

type CreateAssetRequest struct {
	Path     string `json:"path" validate:"required"`
	Filename string `json:"filename" validate:"required,max=255"`
	MimeType string `json:"mime_type" validate:"required,max=100"`
	Size     int64  `json:"size" validate:"required,min=0"`
}

type AssetResponse struct {
	ID        uuid.UUID `json:"id"`
	Path      string    `json:"path"`
	URL       string    `json:"url"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type AssetFilterQuery struct {
	MimeType string `query:"mime_type"`
	Search   string `query:"search"`
}
