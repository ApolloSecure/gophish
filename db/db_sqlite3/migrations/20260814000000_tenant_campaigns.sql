-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "campaigns" ADD COLUMN "tenant_id" VARCHAR(255) NULL;
CREATE INDEX "campaigns_user_tenant_id_idx" ON "campaigns" ("user_id", "tenant_id");

-- +goose Down
DROP INDEX "campaigns_user_tenant_id_idx";

-- SQLite 3.31 (bundled by this project) cannot drop columns directly, so
-- rebuild campaigns without tenant_id.
CREATE TABLE "campaigns_without_tenant" (
    "id" integer primary key autoincrement,
    "user_id" bigint,
    "name" varchar(255) NOT NULL,
    "created_date" datetime,
    "completed_date" datetime,
    "template_id" bigint,
    "page_id" bigint,
    "status" varchar(255),
    "url" varchar(255),
    "smtp_id" bigint,
    "launch_date" DATETIME,
    "send_by_date" DATETIME
);
INSERT INTO "campaigns_without_tenant" ("id", "user_id", "name", "created_date", "completed_date", "template_id", "page_id", "status", "url", "smtp_id", "launch_date", "send_by_date")
SELECT "id", "user_id", "name", "created_date", "completed_date", "template_id", "page_id", "status", "url", "smtp_id", "launch_date", "send_by_date" FROM "campaigns";
DROP TABLE "campaigns";
ALTER TABLE "campaigns_without_tenant" RENAME TO "campaigns";
