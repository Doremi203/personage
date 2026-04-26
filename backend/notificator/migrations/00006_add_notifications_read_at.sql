-- +goose Up
-- +goose StatementBegin
alter table notifications
    add column read_at timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table notifications
    drop column read_at;
-- +goose StatementEnd
