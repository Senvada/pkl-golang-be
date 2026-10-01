CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path TEXT NOT NULL,
    filename VARCHAR(255),
    mime_type VARCHAR(100),
    size BIGINT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_assets_created_by ON assets(created_by);
