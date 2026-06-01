-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- PostgreSQL stores timestamptz values as absolute instants, so fresh installs
-- do not require the MySQL-specific timezone conversion.
SELECT 1;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
SELECT 1;
