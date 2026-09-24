-- +goose Up
CREATE TABLE "tenants" (
    "id" VARCHAR(255) PRIMARY KEY,
    "created_date" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE "groups" ADD COLUMN "tenant_id" VARCHAR(255) NULL;
ALTER TABLE "targets" ADD COLUMN "tenant_id" VARCHAR(255) NULL;
ALTER TABLE "email_requests" ADD COLUMN "tenant_id" VARCHAR(255) NULL;

INSERT INTO "tenants" ("id")
SELECT DISTINCT "tenant_id" FROM "campaigns" WHERE "tenant_id" IS NOT NULL;

CREATE INDEX "groups_user_tenant_id_idx" ON "groups" ("user_id", "tenant_id");
CREATE UNIQUE INDEX "targets_tenant_email_idx" ON "targets" ("tenant_id", "email");
CREATE INDEX "email_requests_user_tenant_id_idx" ON "email_requests" ("user_id", "tenant_id");

ALTER TABLE "campaigns" ADD CONSTRAINT "campaigns_tenant_id_fk" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE CASCADE;
ALTER TABLE "groups" ADD CONSTRAINT "groups_tenant_id_fk" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE CASCADE;
ALTER TABLE "targets" ADD CONSTRAINT "targets_tenant_id_fk" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE CASCADE;
ALTER TABLE "email_requests" ADD CONSTRAINT "email_requests_tenant_id_fk" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE CASCADE;

ALTER TABLE "results" ADD CONSTRAINT "results_campaign_id_fk" FOREIGN KEY ("campaign_id") REFERENCES "campaigns" ("id") ON DELETE CASCADE;
ALTER TABLE "events" ADD CONSTRAINT "events_campaign_id_fk" FOREIGN KEY ("campaign_id") REFERENCES "campaigns" ("id") ON DELETE CASCADE;
ALTER TABLE "mail_logs" ADD CONSTRAINT "mail_logs_campaign_id_fk" FOREIGN KEY ("campaign_id") REFERENCES "campaigns" ("id") ON DELETE CASCADE;
ALTER TABLE "group_targets" ADD CONSTRAINT "group_targets_group_id_fk" FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE CASCADE;
ALTER TABLE "group_targets" ADD CONSTRAINT "group_targets_target_id_fk" FOREIGN KEY ("target_id") REFERENCES "targets" ("id") ON DELETE CASCADE;

-- +goose Down
ALTER TABLE "group_targets" DROP CONSTRAINT "group_targets_target_id_fk";
ALTER TABLE "group_targets" DROP CONSTRAINT "group_targets_group_id_fk";
ALTER TABLE "mail_logs" DROP CONSTRAINT "mail_logs_campaign_id_fk";
ALTER TABLE "events" DROP CONSTRAINT "events_campaign_id_fk";
ALTER TABLE "results" DROP CONSTRAINT "results_campaign_id_fk";
ALTER TABLE "email_requests" DROP CONSTRAINT "email_requests_tenant_id_fk";
ALTER TABLE "targets" DROP CONSTRAINT "targets_tenant_id_fk";
ALTER TABLE "groups" DROP CONSTRAINT "groups_tenant_id_fk";
ALTER TABLE "campaigns" DROP CONSTRAINT "campaigns_tenant_id_fk";
DROP INDEX "email_requests_user_tenant_id_idx";
DROP INDEX "targets_tenant_email_idx";
DROP INDEX "groups_user_tenant_id_idx";
ALTER TABLE "email_requests" DROP COLUMN "tenant_id";
ALTER TABLE "targets" DROP COLUMN "tenant_id";
ALTER TABLE "groups" DROP COLUMN "tenant_id";
DROP TABLE "tenants";
