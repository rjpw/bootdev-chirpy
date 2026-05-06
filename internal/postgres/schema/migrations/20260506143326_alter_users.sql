-- +goose Up
ALTER TABLE users
ADD COLUMN password TEXT NOT NULL DEFAULT 'unset';

-- +goose Down
ALTER TABLE users
ADD COLUMN IF EXISTS password;
