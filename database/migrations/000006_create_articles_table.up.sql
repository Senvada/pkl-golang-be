CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    content JSONB,
    style JSONB,
    cover_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'review', 'reject', 'published', 'archived')),
    published_at TIMESTAMPTZ NULL,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_articles_company_slug_active ON articles(company_id, slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_articles_company_id ON articles(company_id);
CREATE INDEX idx_articles_user_id ON articles(user_id);
CREATE INDEX idx_articles_status_published ON articles(company_id, status, published_at DESC);
