-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "group_targets" ADD COLUMN "custom_fields" TEXT;
ALTER TABLE "results" ADD COLUMN "custom_fields" TEXT;
ALTER TABLE "email_requests" ADD COLUMN "custom_fields" TEXT;

-- +goose Down
-- SQLite migrations do not drop columns to preserve compatibility with older SQLite versions.
