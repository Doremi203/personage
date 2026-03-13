-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS gmail_processing (
    user_id                 UUID PRIMARY KEY,
    last_message_history_id BIGINT,
    created_at              TIMESTAMPTZ DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS gmail_processing;
-- +goose StatementEnd
