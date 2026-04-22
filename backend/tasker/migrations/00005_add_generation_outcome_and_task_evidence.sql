-- +goose Up
-- +goose StatementBegin
ALTER TABLE clusters
    ADD COLUMN IF NOT EXISTS generation_outcome TEXT,
    ADD COLUMN IF NOT EXISTS generation_reason TEXT;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS evidence_event_ids UUID[] NOT NULL DEFAULT '{}'::UUID[];
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
    DROP COLUMN IF EXISTS evidence_event_ids;

ALTER TABLE clusters
    DROP COLUMN IF EXISTS generation_reason,
    DROP COLUMN IF EXISTS generation_outcome;
-- +goose StatementEnd
