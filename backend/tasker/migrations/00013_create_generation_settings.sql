-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS generation_settings
(
    id                          SMALLINT         PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    min_similarity              DOUBLE PRECISION NOT NULL,
    closed_similarity_threshold DOUBLE PRECISION NOT NULL,
    top_k                       INTEGER          NOT NULL,
    max_event_count             INTEGER          NOT NULL,
    inactivity_minutes          INTEGER          NOT NULL,
    batch_size                  INTEGER          NOT NULL,
    updated_at                  TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

INSERT INTO generation_settings (id, min_similarity, closed_similarity_threshold, top_k, max_event_count, inactivity_minutes, batch_size)
VALUES (1, 0.65, 0.90, 5, 5, 5, 10)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS generation_settings;
-- +goose StatementEnd
