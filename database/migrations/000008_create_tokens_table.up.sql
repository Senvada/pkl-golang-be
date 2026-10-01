CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    remember BOOLEAN NOT NULL DEFAULT FALSE,
    user_agent TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_token_hash ON tokens(token_hash);
CREATE INDEX idx_tokens_expires_at ON tokens(expires_at);
