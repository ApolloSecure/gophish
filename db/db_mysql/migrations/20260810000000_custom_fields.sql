-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `group_targets` ADD COLUMN `custom_fields` TEXT;
ALTER TABLE `results` ADD COLUMN `custom_fields` TEXT;
ALTER TABLE `email_requests` ADD COLUMN `custom_fields` TEXT;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `email_requests` DROP COLUMN `custom_fields`;
ALTER TABLE `results` DROP COLUMN `custom_fields`;
ALTER TABLE `group_targets` DROP COLUMN `custom_fields`;
