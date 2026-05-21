-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS manual_moderation_users
(
    user_id    UUID        NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS is_approved BOOLEAN NOT NULL DEFAULT TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
    DROP COLUMN IF EXISTS is_approved;

DROP TABLE IF EXISTS manual_moderation_users;
-- +goose StatementEnd
