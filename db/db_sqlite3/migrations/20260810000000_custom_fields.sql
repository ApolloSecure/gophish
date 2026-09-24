-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "group_targets" ADD COLUMN "custom_fields" TEXT;
ALTER TABLE "results" ADD COLUMN "custom_fields" TEXT;
ALTER TABLE "email_requests" ADD COLUMN "custom_fields" TEXT;

-- +goose Down
-- SQLite 3.31 (bundled by this project) cannot drop columns directly, so
-- rebuild the affected tables without custom_fields.
CREATE TABLE "group_targets_without_custom_fields" (
    "group_id" bigint,
    "target_id" bigint
);
INSERT INTO "group_targets_without_custom_fields" ("group_id", "target_id")
SELECT "group_id", "target_id" FROM "group_targets";
DROP TABLE "group_targets";
ALTER TABLE "group_targets_without_custom_fields" RENAME TO "group_targets";

CREATE TABLE "results_without_custom_fields" (
    "id" integer primary key autoincrement,
    "campaign_id" bigint,
    "user_id" bigint,
    "r_id" varchar(255),
    "email" varchar(255),
    "first_name" varchar(255),
    "last_name" varchar(255),
    "status" varchar(255) NOT NULL,
    "ip" varchar(255),
    "latitude" real,
    "longitude" real,
    "position" VARCHAR(255),
    "send_date" DATETIME,
    "reported" boolean default 0,
    "modified_date" DATETIME
);
INSERT INTO "results_without_custom_fields" ("id", "campaign_id", "user_id", "r_id", "email", "first_name", "last_name", "status", "ip", "latitude", "longitude", "position", "send_date", "reported", "modified_date")
SELECT "id", "campaign_id", "user_id", "r_id", "email", "first_name", "last_name", "status", "ip", "latitude", "longitude", "position", "send_date", "reported", "modified_date" FROM "results";
DROP TABLE "results";
ALTER TABLE "results_without_custom_fields" RENAME TO "results";

CREATE TABLE "email_requests_without_custom_fields" (
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
    "from_address" varchar(255)
);
INSERT INTO "email_requests_without_custom_fields" ("id", "user_id", "template_id", "page_id", "first_name", "last_name", "email", "position", "url", "r_id", "from_address")
SELECT "id", "user_id", "template_id", "page_id", "first_name", "last_name", "email", "position", "url", "r_id", "from_address" FROM "email_requests";
DROP TABLE "email_requests";
ALTER TABLE "email_requests_without_custom_fields" RENAME TO "email_requests";
