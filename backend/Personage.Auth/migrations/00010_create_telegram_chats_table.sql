-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS telegram_chats
(
    user_id    UUID        NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    chat_id    BIGINT      NOT NULL,
    chat_name  TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chat_id)
);
CREATE INDEX IF NOT EXISTS idx_telegram_chats_user_id ON telegram_chats (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_telegram_chats_user_id;
DROP TABLE IF EXISTS telegram_chats;
-- +goose StatementEnd
