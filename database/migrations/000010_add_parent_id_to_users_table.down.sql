DROP INDEX IF EXISTS idx_users_parent_id;
ALTER TABLE users DROP COLUMN IF EXISTS parent_id;
