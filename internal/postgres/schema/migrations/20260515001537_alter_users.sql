-- +goose Up
ALTER TABLE users
ADD COLUMN is_chirpy_red_member boolean DEFAULT false;

-- +goose Down
ALTER TABLE users
ADD COLUMN IF EXISTS is_chirpy_red_member;
