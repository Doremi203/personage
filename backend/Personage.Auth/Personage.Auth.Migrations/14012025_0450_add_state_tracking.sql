ALTER TABLE gmail_token
    ADD COLUMN IF NOT EXISTS last_processed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_gmail_token_last_processed
    ON gmail_token (last_processed_at);