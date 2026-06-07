-- +goose Up
-- +goose StatementBegin
ALTER TABLE gmail_token
    ADD COLUMN failed_refreshes int not null default 0;

ALTER TABLE calendar_token
    ADD COLUMN failed_refreshes int not null default 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE gmail_token
    DROP COLUMN IF EXISTS failed_refreshes;

ALTER TABLE calendar_token
    DROP COLUMN IF EXISTS failed_refreshes;
-- +goose StatementEnd
