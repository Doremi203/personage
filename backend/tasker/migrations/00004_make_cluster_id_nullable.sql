-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ALTER COLUMN cluster_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks ALTER COLUMN cluster_id SET NOT NULL;
-- +goose StatementEnd
