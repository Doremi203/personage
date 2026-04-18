-- +goose Up
-- +goose StatementBegin
alter table notifications
    alter column sent_at drop not null,
    add column status       text        not null default 'sent',
    add column retry_after  timestamptz,
    add column expires_at   timestamptz,
    add column push_payload text,
    add constraint notifications_status_check check (status in ('sent', 'pending', 'dropped'));

create index notifications_ratelimit_idx
    on notifications (recipient_id, type, sent_at)
    where status = 'sent';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists notifications_ratelimit_idx;

-- Remove non-sent rows before restoring the NOT NULL constraint on sent_at.
delete from notifications where status != 'sent';

alter table notifications
    drop constraint if exists notifications_status_check,
    alter column sent_at set not null,
    drop column status,
    drop column retry_after,
    drop column expires_at,
    drop column push_payload;
-- +goose StatementEnd
