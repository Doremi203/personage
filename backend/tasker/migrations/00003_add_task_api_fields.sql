-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN end_time TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN category VARCHAR(20) NOT NULL DEFAULT 'personal';

-- Add full-text search support
ALTER TABLE tasks ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))) STORED;
CREATE INDEX idx_tasks_search ON tasks USING gin(search_vector);

-- Add index for category filtering
CREATE INDEX idx_tasks_category ON tasks (category);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasks_category;
DROP INDEX IF EXISTS idx_tasks_search;
ALTER TABLE tasks DROP COLUMN IF EXISTS search_vector;
ALTER TABLE tasks DROP COLUMN IF EXISTS category;
ALTER TABLE tasks DROP COLUMN IF EXISTS end_time;
-- +goose StatementEnd
