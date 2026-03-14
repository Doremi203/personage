-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS processing_snapshot (
    id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    start  TIMESTAMPTZ NOT NULL,
    finish TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_processing_snapshot_active_period
    ON processing_snapshot (start, finish);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_processing_snapshot_active_period;
DROP TABLE IF EXISTS processing_snapshot;
-- +goose StatementEnd
