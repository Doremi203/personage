-- +goose Up
-- +goose StatementBegin
alter table notifications
    alter column sent_at drop not null,
    add column status       text        not null default 'sent',
    add column retry_after  timestamptz,
    add column expires_at   timestamptz,
    add column push_payload text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table notifications
    alter column sent_at set not null,
    drop column status,
    drop column retry_after,
    drop column expires_at,
    drop column push_payload;
-- +goose StatementEnd
