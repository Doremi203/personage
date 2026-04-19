-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS calendar_token
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    last_processed_at TIMESTAMPTZ,
    status        SMALLINT NOT NULL,
    UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_calendar_token_last_processed
    ON calendar_token (last_processed_at);
CREATE INDEX IF NOT EXISTS idx_calendar_token_user_id ON calendar_token (user_id);
CREATE INDEX IF NOT EXISTS idx_calendar_token_expires_at ON calendar_token (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_gmail_token_last_processed;
DROP INDEX IF EXISTS idx_calendar_token_user_id;
DROP INDEX IF EXISTS idx_calendar_token_expires_at;
DROP TABLE IF EXISTS calendar_token;
-- +goose StatementEnd
    