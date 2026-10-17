ALTER TABLE users ADD COLUMN parent_id UUID NULL;
CREATE INDEX idx_users_parent_id ON users(parent_id);
