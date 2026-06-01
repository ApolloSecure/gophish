
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "users" ADD COLUMN account_locked BOOLEAN DEFAULT false;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
