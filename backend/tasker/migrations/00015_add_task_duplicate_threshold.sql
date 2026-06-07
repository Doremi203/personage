-- +goose Up
-- +goose StatementBegin
ALTER TABLE generation_settings
    ADD COLUMN IF NOT EXISTS task_duplicate_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.97;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE generation_settings
    DROP COLUMN IF EXISTS task_duplicate_threshold;
-- +goose StatementEnd
