-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS gmail_token
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    gmail_email   TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_gmail_token_user_id ON gmail_token (user_id);
CREATE INDEX IF NOT EXISTS idx_gmail_token_expires_at ON gmail_token (expires_at);
CREATE INDEX IF NOT EXISTS idx_gmail_token_email ON gmail_token (gmail_email);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_gmail_token_email;
DROP INDEX IF EXISTS idx_gmail_token_expires_at;
DROP INDEX IF EXISTS idx_gmail_token_user_id;
DROP TABLE IF EXISTS gmail_token;
-- +goose StatementEnd
