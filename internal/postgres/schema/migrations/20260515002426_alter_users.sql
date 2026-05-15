-- +goose Up
ALTER TABLE users
ALTER COLUMN is_chirpy_red_member SET NOT NULL;

-- +goose Down
ALTER TABLE users
ALTER COLUMN is_chirpy_red_member DROP NOT NULL;
