-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "users" ADD COLUMN last_login timestamp with time zone;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
