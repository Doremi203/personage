-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS processing_pauses
(
    user_id      UUID        NOT NULL PRIMARY KEY,
    paused_until TIMESTAMPTZ,
    reason       TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS processing_pauses;
-- +goose StatementEnd
