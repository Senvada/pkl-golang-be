CREATE TABLE IF NOT EXISTS users_departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_departments_unique ON users_departments(user_id, department_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_departments_user_id ON users_departments(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_departments_department_id ON users_departments(department_id) WHERE deleted_at IS NULL;

-- Backfill from existing users.department_id
INSERT INTO users_departments (user_id, department_id)
SELECT id, department_id FROM users 
WHERE department_id IS NOT NULL AND deleted_at IS NULL
ON CONFLICT (user_id, department_id) WHERE deleted_at IS NULL DO NOTHING;
