CREATE TABLE gmail_token (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    access_token    TEXT NOT NULL,
    refresh_token   TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    gmail_email     TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_gmail_token_user_id ON gmail_token(user_id);
CREATE INDEX idx_gmail_token_expires_at ON gmail_token(expires_at);
CREATE INDEX idx_gmail_token_email ON gmail_token(gmail_email);