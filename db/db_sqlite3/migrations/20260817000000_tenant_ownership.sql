-- +goose Up
CREATE TABLE "tenants" (
    "id" VARCHAR(255) PRIMARY KEY,
    "created_date" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE "groups" ADD COLUMN "tenant_id" VARCHAR(255) NULL;
ALTER TABLE "targets" ADD COLUMN "tenant_id" VARCHAR(255) NULL;
ALTER TABLE "email_requests" ADD COLUMN "tenant_id" VARCHAR(255) NULL;

INSERT OR IGNORE INTO "tenants" ("id")
SELECT DISTINCT "tenant_id" FROM "campaigns" WHERE "tenant_id" IS NOT NULL;

CREATE INDEX "groups_user_tenant_id_idx" ON "groups" ("user_id", "tenant_id");
CREATE UNIQUE INDEX "targets_tenant_email_idx" ON "targets" ("tenant_id", "email");
CREATE INDEX "email_requests_user_tenant_id_idx" ON "email_requests" ("user_id", "tenant_id");

CREATE TRIGGER "campaigns_tenant_insert_fk" BEFORE INSERT ON "campaigns" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'campaign tenant does not exist'); END;
CREATE TRIGGER "campaigns_tenant_update_fk" BEFORE UPDATE OF "tenant_id" ON "campaigns" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'campaign tenant does not exist'); END;
CREATE TRIGGER "groups_tenant_insert_fk" BEFORE INSERT ON "groups" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'group tenant does not exist'); END;
CREATE TRIGGER "groups_tenant_update_fk" BEFORE UPDATE OF "tenant_id" ON "groups" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'group tenant does not exist'); END;
CREATE TRIGGER "groups_tenant_relationship_update" BEFORE UPDATE OF "tenant_id" ON "groups" WHEN EXISTS (SELECT 1 FROM "group_targets" gt JOIN "targets" t ON t."id" = gt."target_id" WHERE gt."group_id" = NEW."id" AND COALESCE(t."tenant_id", '') <> COALESCE(NEW."tenant_id", '')) BEGIN SELECT RAISE(ABORT, 'group target tenant mismatch'); END;
CREATE TRIGGER "targets_tenant_insert_fk" BEFORE INSERT ON "targets" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'target tenant does not exist'); END;
CREATE TRIGGER "targets_tenant_update_fk" BEFORE UPDATE OF "tenant_id" ON "targets" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'target tenant does not exist'); END;
CREATE TRIGGER "targets_tenant_relationship_update" BEFORE UPDATE OF "tenant_id" ON "targets" WHEN EXISTS (SELECT 1 FROM "group_targets" gt JOIN "groups" g ON g."id" = gt."group_id" WHERE gt."target_id" = NEW."id" AND COALESCE(g."tenant_id", '') <> COALESCE(NEW."tenant_id", '')) BEGIN SELECT RAISE(ABORT, 'group target tenant mismatch'); END;
CREATE TRIGGER "email_requests_tenant_insert_fk" BEFORE INSERT ON "email_requests" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'email request tenant does not exist'); END;
CREATE TRIGGER "email_requests_tenant_update_fk" BEFORE UPDATE OF "tenant_id" ON "email_requests" WHEN NEW."tenant_id" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "tenants" WHERE "id" = NEW."tenant_id") BEGIN SELECT RAISE(ABORT, 'email request tenant does not exist'); END;

CREATE TRIGGER "results_campaign_insert_fk" BEFORE INSERT ON "results" WHEN NOT EXISTS (SELECT 1 FROM "campaigns" WHERE "id" = NEW."campaign_id") BEGIN SELECT RAISE(ABORT, 'result campaign does not exist'); END;
CREATE TRIGGER "results_campaign_update_fk" BEFORE UPDATE OF "campaign_id" ON "results" WHEN NOT EXISTS (SELECT 1 FROM "campaigns" WHERE "id" = NEW."campaign_id") BEGIN SELECT RAISE(ABORT, 'result campaign does not exist'); END;
CREATE TRIGGER "events_campaign_insert_fk" BEFORE INSERT ON "events" WHEN NOT EXISTS (SELECT 1 FROM "campaigns" WHERE "id" = NEW."campaign_id") BEGIN SELECT RAISE(ABORT, 'event campaign does not exist'); END;
CREATE TRIGGER "events_campaign_update_fk" BEFORE UPDATE OF "campaign_id" ON "events" WHEN NOT EXISTS (SELECT 1 FROM "campaigns" WHERE "id" = NEW."campaign_id") BEGIN SELECT RAISE(ABORT, 'event campaign does not exist'); END;
CREATE TRIGGER "mail_logs_campaign_insert_fk" BEFORE INSERT ON "mail_logs" WHEN NOT EXISTS (SELECT 1 FROM "campaigns" WHERE "id" = NEW."campaign_id") BEGIN SELECT RAISE(ABORT, 'mail log campaign does not exist'); END;
CREATE TRIGGER "mail_logs_campaign_update_fk" BEFORE UPDATE OF "campaign_id" ON "mail_logs" WHEN NOT EXISTS (SELECT 1 FROM "campaigns" WHERE "id" = NEW."campaign_id") BEGIN SELECT RAISE(ABORT, 'mail log campaign does not exist'); END;
CREATE TRIGGER "group_targets_insert_fk" BEFORE INSERT ON "group_targets" WHEN NOT EXISTS (SELECT 1 FROM "groups" WHERE "id" = NEW."group_id") OR NOT EXISTS (SELECT 1 FROM "targets" WHERE "id" = NEW."target_id") BEGIN SELECT RAISE(ABORT, 'group target parent does not exist'); END;
CREATE TRIGGER "group_targets_update_fk" BEFORE UPDATE OF "group_id", "target_id" ON "group_targets" WHEN NOT EXISTS (SELECT 1 FROM "groups" WHERE "id" = NEW."group_id") OR NOT EXISTS (SELECT 1 FROM "targets" WHERE "id" = NEW."target_id") BEGIN SELECT RAISE(ABORT, 'group target parent does not exist'); END;
CREATE TRIGGER "group_targets_tenant_match" BEFORE INSERT ON "group_targets" WHEN (SELECT COALESCE("tenant_id", '') FROM "groups" WHERE "id" = NEW."group_id") <> (SELECT COALESCE("tenant_id", '') FROM "targets" WHERE "id" = NEW."target_id") BEGIN SELECT RAISE(ABORT, 'group target tenant mismatch'); END;
CREATE TRIGGER "group_targets_tenant_match_update" BEFORE UPDATE OF "group_id", "target_id" ON "group_targets" WHEN (SELECT COALESCE("tenant_id", '') FROM "groups" WHERE "id" = NEW."group_id") <> (SELECT COALESCE("tenant_id", '') FROM "targets" WHERE "id" = NEW."target_id") BEGIN SELECT RAISE(ABORT, 'group target tenant mismatch'); END;

CREATE TRIGGER "tenants_delete_cascade" AFTER DELETE ON "tenants" BEGIN DELETE FROM "campaigns" WHERE "tenant_id" = OLD."id"; DELETE FROM "groups" WHERE "tenant_id" = OLD."id"; DELETE FROM "targets" WHERE "tenant_id" = OLD."id"; DELETE FROM "email_requests" WHERE "tenant_id" = OLD."id"; END;
CREATE TRIGGER "campaigns_delete_cascade" AFTER DELETE ON "campaigns" BEGIN DELETE FROM "results" WHERE "campaign_id" = OLD."id"; DELETE FROM "events" WHERE "campaign_id" = OLD."id"; DELETE FROM "mail_logs" WHERE "campaign_id" = OLD."id"; END;
CREATE TRIGGER "groups_delete_cascade" AFTER DELETE ON "groups" BEGIN DELETE FROM "group_targets" WHERE "group_id" = OLD."id"; END;
CREATE TRIGGER "targets_delete_cascade" AFTER DELETE ON "targets" BEGIN DELETE FROM "group_targets" WHERE "target_id" = OLD."id"; END;

-- +goose Down
DROP TRIGGER "targets_delete_cascade";
DROP TRIGGER "groups_delete_cascade";
DROP TRIGGER "campaigns_delete_cascade";
DROP TRIGGER "tenants_delete_cascade";
DROP TRIGGER "group_targets_tenant_match_update";
DROP TRIGGER "group_targets_tenant_match";
DROP TRIGGER "group_targets_update_fk";
DROP TRIGGER "group_targets_insert_fk";
DROP TRIGGER "mail_logs_campaign_update_fk";
DROP TRIGGER "mail_logs_campaign_insert_fk";
DROP TRIGGER "events_campaign_update_fk";
DROP TRIGGER "events_campaign_insert_fk";
DROP TRIGGER "results_campaign_update_fk";
DROP TRIGGER "results_campaign_insert_fk";
DROP TRIGGER "email_requests_tenant_update_fk";
DROP TRIGGER "email_requests_tenant_insert_fk";
DROP TRIGGER "targets_tenant_relationship_update";
DROP TRIGGER "targets_tenant_update_fk";
DROP TRIGGER "targets_tenant_insert_fk";
DROP TRIGGER "groups_tenant_relationship_update";
DROP TRIGGER "groups_tenant_update_fk";
DROP TRIGGER "groups_tenant_insert_fk";
DROP TRIGGER "campaigns_tenant_update_fk";
DROP TRIGGER "campaigns_tenant_insert_fk";
DROP INDEX "email_requests_user_tenant_id_idx";
DROP INDEX "targets_tenant_email_idx";
DROP INDEX "groups_user_tenant_id_idx";
DROP TABLE "tenants";

-- SQLite 3.31 (bundled by this project) cannot drop columns directly, so
-- rebuild the affected tables without tenant_id. This preserves all columns
-- that existed before this migration, including custom_fields.
CREATE TABLE "groups_without_tenant" (
    "id" integer primary key autoincrement,
    "user_id" bigint,
    "name" varchar(255),
    "modified_date" datetime
);
INSERT INTO "groups_without_tenant" ("id", "user_id", "name", "modified_date")
SELECT "id", "user_id", "name", "modified_date" FROM "groups";
DROP TABLE "groups";
ALTER TABLE "groups_without_tenant" RENAME TO "groups";

CREATE TABLE "targets_without_tenant" (
    "id" integer primary key autoincrement,
    "first_name" varchar(255),
    "last_name" varchar(255),
    "email" varchar(255),
    "position" varchar(255)
);
INSERT INTO "targets_without_tenant" ("id", "first_name", "last_name", "email", "position")
SELECT "id", "first_name", "last_name", "email", "position" FROM "targets";
DROP TABLE "targets";
ALTER TABLE "targets_without_tenant" RENAME TO "targets";

CREATE TABLE "email_requests_without_tenant" (
    "id" integer primary key autoincrement,
    "user_id" integer,
    "template_id" integer,
    "page_id" integer,
    "first_name" varchar(255),
    "last_name" varchar(255),
    "email" varchar(255),
    "position" varchar(255),
    "url" varchar(255),
    "r_id" varchar(255),
    "from_address" varchar(255),
    "custom_fields" TEXT
);
INSERT INTO "email_requests_without_tenant" ("id", "user_id", "template_id", "page_id", "first_name", "last_name", "email", "position", "url", "r_id", "from_address", "custom_fields")
SELECT "id", "user_id", "template_id", "page_id", "first_name", "last_name", "email", "position", "url", "r_id", "from_address", "custom_fields" FROM "email_requests";
DROP TABLE "email_requests";
ALTER TABLE "email_requests_without_tenant" RENAME TO "email_requests";
