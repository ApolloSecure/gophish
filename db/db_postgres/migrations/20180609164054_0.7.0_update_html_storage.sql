-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "templates" ALTER COLUMN "html" TYPE text;
ALTER TABLE "pages" ALTER COLUMN "html" TYPE text;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE "templates" ALTER COLUMN "html" TYPE text;
ALTER TABLE "pages" ALTER COLUMN "html" TYPE text;
