-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS calendar_processing (
    user_id                     UUID PRIMARY KEY,
    last_sync_token             TEXT,
    last_event_updated_time     TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW()
);

-- Index for finding users that need processing based on last update time
CREATE INDEX IF NOT EXISTS idx_calendar_processing_updated
ON calendar_processing(last_event_updated_time)
WHERE last_event_updated_time IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_calendar_processing_sync_token
ON calendar_processing(last_sync_token)
WHERE last_sync_token IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_calendar_processing_sync_token;
DROP INDEX IF EXISTS idx_calendar_processing_updated;
DROP TABLE IF EXISTS calendar_processing;
-- +goose StatementEnd
