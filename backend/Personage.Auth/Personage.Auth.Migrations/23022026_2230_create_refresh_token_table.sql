CREATE TABLE IF NOT EXISTS refresh_token (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token       TEXT UNIQUE,
    user_id     UUID references "user"(id) on delete cascade,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    expires_at  TIMESTAMPTZ not null
);
