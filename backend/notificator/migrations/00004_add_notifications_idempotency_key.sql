-- +goose Up
-- +goose StatementBegin
ALTER TABLE notifications ADD COLUMN idempotency_key text;

CREATE UNIQUE INDEX notifications_idempotency_key_uniq
    ON notifications (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS notifications_idempotency_key_uniq;
ALTER TABLE notifications DROP COLUMN IF EXISTS idempotency_key;
-- +goose StatementEnd
