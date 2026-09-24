-- +goose Up
CREATE TABLE `tenants` (
    `id` VARCHAR(255) PRIMARY KEY,
    `created_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

ALTER TABLE `groups` ADD COLUMN `tenant_id` VARCHAR(255) NULL;
ALTER TABLE `targets` ADD COLUMN `tenant_id` VARCHAR(255) NULL;
ALTER TABLE `email_requests` ADD COLUMN `tenant_id` VARCHAR(255) NULL;

INSERT IGNORE INTO `tenants` (`id`)
SELECT DISTINCT `tenant_id` FROM `campaigns` WHERE `tenant_id` IS NOT NULL;

CREATE INDEX `groups_user_tenant_id_idx` ON `groups` (`user_id`, `tenant_id`);
CREATE UNIQUE INDEX `targets_tenant_email_idx` ON `targets` (`tenant_id`, `email`);
CREATE INDEX `email_requests_user_tenant_id_idx` ON `email_requests` (`user_id`, `tenant_id`);

ALTER TABLE `campaigns` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE `groups` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE `targets` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE `mail_logs` MODIFY COLUMN `campaign_id` BIGINT NULL;

ALTER TABLE `campaigns` ADD CONSTRAINT `campaigns_tenant_id_fk` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE;
ALTER TABLE `groups` ADD CONSTRAINT `groups_tenant_id_fk` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE;
ALTER TABLE `targets` ADD CONSTRAINT `targets_tenant_id_fk` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE;
ALTER TABLE `email_requests` ADD CONSTRAINT `email_requests_tenant_id_fk` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE;

ALTER TABLE `results` ADD CONSTRAINT `results_campaign_id_fk` FOREIGN KEY (`campaign_id`) REFERENCES `campaigns` (`id`) ON DELETE CASCADE;
ALTER TABLE `events` ADD CONSTRAINT `events_campaign_id_fk` FOREIGN KEY (`campaign_id`) REFERENCES `campaigns` (`id`) ON DELETE CASCADE;
ALTER TABLE `mail_logs` ADD CONSTRAINT `mail_logs_campaign_id_fk` FOREIGN KEY (`campaign_id`) REFERENCES `campaigns` (`id`) ON DELETE CASCADE;
ALTER TABLE `group_targets` ADD CONSTRAINT `group_targets_group_id_fk` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`) ON DELETE CASCADE;
ALTER TABLE `group_targets` ADD CONSTRAINT `group_targets_target_id_fk` FOREIGN KEY (`target_id`) REFERENCES `targets` (`id`) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE `group_targets` DROP FOREIGN KEY `group_targets_target_id_fk`;
ALTER TABLE `group_targets` DROP FOREIGN KEY `group_targets_group_id_fk`;
ALTER TABLE `mail_logs` DROP FOREIGN KEY `mail_logs_campaign_id_fk`;
ALTER TABLE `events` DROP FOREIGN KEY `events_campaign_id_fk`;
ALTER TABLE `results` DROP FOREIGN KEY `results_campaign_id_fk`;
ALTER TABLE `email_requests` DROP FOREIGN KEY `email_requests_tenant_id_fk`;
ALTER TABLE `targets` DROP FOREIGN KEY `targets_tenant_id_fk`;
ALTER TABLE `groups` DROP FOREIGN KEY `groups_tenant_id_fk`;
ALTER TABLE `campaigns` DROP FOREIGN KEY `campaigns_tenant_id_fk`;
DROP INDEX `email_requests_user_tenant_id_idx` ON `email_requests`;
DROP INDEX `targets_tenant_email_idx` ON `targets`;
DROP INDEX `groups_user_tenant_id_idx` ON `groups`;
ALTER TABLE `email_requests` DROP COLUMN `tenant_id`;
ALTER TABLE `targets` DROP COLUMN `tenant_id`;
ALTER TABLE `groups` DROP COLUMN `tenant_id`;
DROP TABLE `tenants`;
