
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "mail_logs" (
    "id" bigserial primary key,
    "campaign_id" integer,
    "user_id" integer,
    "send_date" timestamp with time zone,
    "send_attempt" integer,
    "r_id" varchar(255),
    "processing" boolean);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "mail_logs";
