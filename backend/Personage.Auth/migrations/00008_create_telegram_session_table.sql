-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS telegram_session
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID UNIQUE NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    session    TEXT      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_processed_at timestamptz NULL
);

CREATE INDEX IF NOT EXISTS idx_telegram_session_user_id ON telegram_session (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_telegram_session_user_id;
DROP TABLE IF EXISTS telegram_session;
-- +goose StatementEnd
