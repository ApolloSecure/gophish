-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "campaigns" ADD COLUMN "tenant_id" VARCHAR(255) NULL;
CREATE INDEX "campaigns_user_tenant_id_idx" ON "campaigns" ("user_id", "tenant_id");

-- +goose Down
-- SQLite migrations retain the column for compatibility with older SQLite versions.
DROP INDEX "campaigns_user_tenant_id_idx";
