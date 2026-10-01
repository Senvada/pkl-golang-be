CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL,
    department_id UUID,
    company_id UUID,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password TEXT NOT NULL,
    avatar_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'waiting', 'inactive', 'banned')),
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role_id ON users(role_id);
CREATE INDEX idx_users_department_id ON users(department_id);
CREATE INDEX idx_users_company_id ON users(company_id);
CREATE INDEX idx_users_status ON users(status);
