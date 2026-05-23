-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN date DATE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks DROP COLUMN date;
-- +goose StatementEnd
