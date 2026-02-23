-- +goose Up
-- +goose StatementBegin
-- Create clusters table
CREATE TABLE IF NOT EXISTS clusters
(
    cluster_id  UUID PRIMARY KEY,
    user_id     UUID         NOT NULL,
    centroid    vector(1536) NOT NULL,
    event_count INTEGER      NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    status      VARCHAR(20)  NOT NULL
);

CREATE INDEX idx_clusters_user_id ON clusters (user_id);

CREATE INDEX idx_clusters_status ON clusters (status);

-- Index for efficient similarity search on centroids
CREATE INDEX idx_clusters_centroid_hnsw ON clusters
    USING hnsw (centroid vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Create events table
CREATE TABLE IF NOT EXISTS events
(
    event_id    UUID PRIMARY KEY,
    user_id     UUID         NOT NULL,
    source      VARCHAR(50)  NOT NULL,
    occurred_at TIMESTAMPTZ  NOT NULL,
    context     text         NOT NULL,
    embedding   vector(1536) NOT NULL,
    cluster_id  UUID         NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_events_cluster_id FOREIGN KEY (cluster_id) REFERENCES clusters (cluster_id)
);

CREATE INDEX idx_events_user_occurred ON events (user_id, occurred_at DESC);

CREATE INDEX idx_events_user_source ON events (user_id, source, occurred_at DESC);

CREATE INDEX idx_events_cluster_id ON events (cluster_id);

CREATE INDEX idx_events_embedding_hnsw ON events
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS clusters;
-- +goose StatementEnd
