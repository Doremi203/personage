-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
    DROP COLUMN IF EXISTS evidence_event_ids;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS embedding vector(1536);

CREATE INDEX IF NOT EXISTS idx_tasks_embedding ON tasks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasks_embedding;

ALTER TABLE tasks
    DROP COLUMN IF EXISTS embedding;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS evidence_event_ids UUID[] NOT NULL DEFAULT '{}'::UUID[];
-- +goose StatementEnd
