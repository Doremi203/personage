-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS telegram_seen_message (
    user_id     UUID        NOT NULL,
    chat_id     BIGINT      NOT NULL,
    message_id  BIGINT      NOT NULL,
    seen_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chat_id, message_id)
);

-- Index supports TTL cleanup by seen_at.
CREATE INDEX IF NOT EXISTS idx_telegram_seen_message_seen_at
ON telegram_seen_message (seen_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_telegram_seen_message_seen_at;
DROP TABLE IF EXISTS telegram_seen_message;
-- +goose StatementEnd
