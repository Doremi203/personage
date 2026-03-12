-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS refresh_token
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token      TEXT UNIQUE,
    user_id    UUID REFERENCES "user" (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS refresh_token;
-- +goose StatementEnd
