
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "imap" ADD COLUMN ignore_cert_errors BOOLEAN DEFAULT false;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
