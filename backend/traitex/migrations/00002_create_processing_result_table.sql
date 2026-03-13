-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS processing_result (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    processed_at TIMESTAMPTZ NOT NULL,
    event        JSONB NOT NULL
);

CREATE INDEX idx_processing_result_processed_at
    ON processing_result (processed_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_processing_result_processed_at;
DROP TABLE IF EXISTS processing_result;
-- +goose StatementEnd
