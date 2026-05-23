-- +goose Up
-- +goose StatementBegin
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS similarity FLOAT8;

UPDATE events e
SET similarity = 1 - (e.embedding <=> c.centroid)
FROM clusters c
WHERE e.cluster_id = c.cluster_id
  AND e.similarity IS NULL;

ALTER TABLE events
    ALTER COLUMN similarity SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE events
    DROP COLUMN IF EXISTS similarity;
-- +goose StatementEnd
