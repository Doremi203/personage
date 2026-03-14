-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS telegram_processing
(
    user_id         UUID PRIMARY KEY,
    last_message_id BIGINT NOT NULL,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS telegram_processing;
-- +goose StatementEnd
