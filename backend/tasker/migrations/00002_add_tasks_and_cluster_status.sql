-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasks
(
    task_id          UUID PRIMARY KEY,
    user_id          UUID        NOT NULL,
    cluster_id       UUID        NOT NULL UNIQUE,
    title            TEXT        NOT NULL,
    description      TEXT,
    duration_minutes INTEGER,
    priority         INTEGER,
    deadline         TIMESTAMPTZ,
    start_time       TIMESTAMPTZ,
    status           VARCHAR(20) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_tasks_cluster_id FOREIGN KEY (cluster_id) REFERENCES clusters (cluster_id)
);

CREATE INDEX idx_tasks_user_id ON tasks (user_id);
CREATE INDEX idx_tasks_status ON tasks (status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS tasks;

-- +goose StatementEnd
