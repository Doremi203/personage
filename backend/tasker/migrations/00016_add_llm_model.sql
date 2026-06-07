-- +goose Up
-- +goose StatementBegin
ALTER TABLE generation_settings
    ADD COLUMN IF NOT EXISTS llm_model TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE generation_settings
    DROP COLUMN IF EXISTS llm_model;
-- +goose StatementEnd
