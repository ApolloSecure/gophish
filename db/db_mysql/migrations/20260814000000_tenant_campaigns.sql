-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `campaigns` ADD COLUMN `tenant_id` VARCHAR(255) NULL;
CREATE INDEX `campaigns_user_tenant_id_idx` ON `campaigns` (`user_id`, `tenant_id`);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP INDEX `campaigns_user_tenant_id_idx` ON `campaigns`;
ALTER TABLE `campaigns` DROP COLUMN `tenant_id`;
