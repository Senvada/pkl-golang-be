CREATE TABLE user_google_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    google_id VARCHAR(255) NOT NULL,
    google_email VARCHAR(255) NOT NULL,
    google_name VARCHAR(255),
    google_avatar_url TEXT,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_user_google_accounts_user_id ON user_google_accounts(user_id);
CREATE UNIQUE INDEX idx_user_google_accounts_google_id ON user_google_accounts(google_id);
