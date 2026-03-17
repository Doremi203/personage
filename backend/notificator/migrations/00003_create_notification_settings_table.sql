-- +goose Up
-- +goose StatementBegin
create table if not exists notification_settings
(
    recipient_id uuid not null,
    type         text not null,
    enabled      bool not null default true,
    primary key (recipient_id, type)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists notification_settings;
-- +goose StatementEnd
