CREATE TABLE password_reset_token
(
    id         UUID PRIMARY KEY,
    user_id    UUID      NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    token      TEXT      NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    used_at    TIMESTAMP
);

CREATE INDEX idx_password_reset_tokens_token ON password_reset_token (token);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_token (user_id);