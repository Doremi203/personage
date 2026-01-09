CREATE TABLE oauth_state (
    state           TEXT PRIMARY KEY,
    user_email      TEXT NOT NULL,
    redirect_uri    TEXT NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_state_expires ON oauth_state(expires_at);