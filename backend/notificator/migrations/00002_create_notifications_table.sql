-- +goose Up
-- +goose StatementBegin
create table if not exists notifications
(
    id           uuid primary key default gen_random_uuid(),
    recipient_id uuid        not null,
    title        text        not null,
    type         text        not null,
    text         text        not null,
    sent_at      timestamptz not null default now()
);

create index notifications_recipient_id_sent_at_idx on notifications (recipient_id, sent_at desc);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists notifications;
-- +goose StatementEnd
