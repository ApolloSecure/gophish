-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "results" ADD COLUMN "modified_date" timestamp with time zone;

UPDATE "results"
SET "modified_date" = event_times.max_time
FROM (
    SELECT "email", "campaign_id", max("time") AS max_time
    FROM "events"
    GROUP BY "email", "campaign_id"
) AS event_times
WHERE "results"."email" = event_times."email"
  AND "results"."campaign_id" = event_times."campaign_id";



-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
