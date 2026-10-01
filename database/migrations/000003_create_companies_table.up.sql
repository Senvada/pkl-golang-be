CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone_number VARCHAR(30),
    address TEXT,
    description TEXT,
    content JSONB,
    image_url TEXT,
    gmaps_link TEXT,
    lat DECIMAL(10,8),
    lng DECIMAL(11,8),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_companies_slug_active ON companies(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_companies_status ON companies(status);
