package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

// Tenant is the ownership root for customer-specific phishing data.
type Tenant struct {
	Id          string    `json:"id" gorm:"column:id;primary_key;type:varchar(255)"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

// TenantDeletionCounts contains non-PII counts for a tenant purge.
type TenantDeletionCounts struct {
	Campaigns     int64 `json:"campaigns"`
	Results       int64 `json:"results"`
	Events        int64 `json:"events"`
	MailLogs      int64 `json:"mail_logs"`
	Groups        int64 `json:"groups"`
	GroupTargets  int64 `json:"group_targets"`
	Targets       int64 `json:"targets"`
	EmailRequests int64 `json:"email_requests"`
	Tenant        int64 `json:"tenant"`
}

// TenantPurgeResult is returned after an idempotent tenant purge.
type TenantPurgeResult struct {
	TenantId string               `json:"tenant_id"`
	Deleted  TenantDeletionCounts `json:"deleted"`
}

func ensureTenant(tx *gorm.DB, tenantID *string) error {
	if err := ValidateTenantID(tenantID); err != nil {
		return err
	}
	if tenantID == nil {
		return nil
	}
	createdDate := time.Now().UTC()
	switch conf.DBName {
	case "postgres":
		return tx.Exec("INSERT INTO tenants (id, created_date) VALUES (?, ?) ON CONFLICT (id) DO NOTHING", *tenantID, createdDate).Error
	case "mysql":
		return tx.Exec("INSERT IGNORE INTO tenants (id, created_date) VALUES (?, ?)", *tenantID, createdDate).Error
	default:
		return tx.Exec("INSERT OR IGNORE INTO tenants (id, created_date) VALUES (?, ?)", *tenantID, createdDate).Error
	}
}

// PurgeTenant deletes all directly and indirectly tenant-owned data in one
// transaction. Unknown and already-purged tenants return zero counts.
func PurgeTenant(tenantID string) (TenantPurgeResult, error) {
	return purgeTenant(tenantID, nil)
}

// purgeTenant accepts a test hook that runs after child deletions and before
// parent deletions, allowing rollback behaviour to be verified.
func purgeTenant(tenantID string, failureHook func(*gorm.DB) error) (TenantPurgeResult, error) {
	result := TenantPurgeResult{TenantId: tenantID}
	if err := ValidateTenantID(&tenantID); err != nil {
		return result, err
	}

	tx := db.Begin()
	if tx.Error != nil {
		return result, tx.Error
	}
	rollback := func(err error) (TenantPurgeResult, error) {
		tx.Rollback()
		return TenantPurgeResult{TenantId: tenantID}, err
	}

	campaignIDs := []int64{}
	if err := tx.Table("campaigns").Where("tenant_id = ?", tenantID).Pluck("id", &campaignIDs).Error; err != nil {
		return rollback(err)
	}
	if len(campaignIDs) > 0 {
		deleted := tx.Where("campaign_id IN (?)", campaignIDs).Delete(&Result{})
		if deleted.Error != nil {
			return rollback(deleted.Error)
		}
		result.Deleted.Results = deleted.RowsAffected

		deleted = tx.Where("campaign_id IN (?)", campaignIDs).Delete(&Event{})
		if deleted.Error != nil {
			return rollback(deleted.Error)
		}
		result.Deleted.Events = deleted.RowsAffected

		deleted = tx.Where("campaign_id IN (?)", campaignIDs).Delete(&MailLog{})
		if deleted.Error != nil {
			return rollback(deleted.Error)
		}
		result.Deleted.MailLogs = deleted.RowsAffected
	}

	groupIDs := []int64{}
	if err := tx.Table("groups").Where("tenant_id = ?", tenantID).Pluck("id", &groupIDs).Error; err != nil {
		return rollback(err)
	}
	targetIDs := []int64{}
	if err := tx.Table("targets").Where("tenant_id = ?", tenantID).Pluck("id", &targetIDs).Error; err != nil {
		return rollback(err)
	}
	if len(groupIDs) > 0 {
		deleted := tx.Where("group_id IN (?)", groupIDs).Delete(&GroupTarget{})
		if deleted.Error != nil {
			return rollback(deleted.Error)
		}
		result.Deleted.GroupTargets += deleted.RowsAffected
	}
	if len(targetIDs) > 0 {
		deleted := tx.Where("target_id IN (?)", targetIDs).Delete(&GroupTarget{})
		if deleted.Error != nil {
			return rollback(deleted.Error)
		}
		result.Deleted.GroupTargets += deleted.RowsAffected
	}

	if failureHook != nil {
		if err := failureHook(tx); err != nil {
			return rollback(err)
		}
	}

	deleted := tx.Where("tenant_id = ?", tenantID).Delete(&Campaign{})
	if deleted.Error != nil {
		return rollback(deleted.Error)
	}
	result.Deleted.Campaigns = deleted.RowsAffected

	deleted = tx.Where("tenant_id = ?", tenantID).Delete(&Group{})
	if deleted.Error != nil {
		return rollback(deleted.Error)
	}
	result.Deleted.Groups = deleted.RowsAffected

	deleted = tx.Where("tenant_id = ?", tenantID).Delete(&Target{})
	if deleted.Error != nil {
		return rollback(deleted.Error)
	}
	result.Deleted.Targets = deleted.RowsAffected

	deleted = tx.Where("tenant_id = ?", tenantID).Delete(&EmailRequest{})
	if deleted.Error != nil {
		return rollback(deleted.Error)
	}
	result.Deleted.EmailRequests = deleted.RowsAffected

	deleted = tx.Where("id = ?", tenantID).Delete(&Tenant{})
	if deleted.Error != nil {
		return rollback(deleted.Error)
	}
	result.Deleted.Tenant = deleted.RowsAffected

	if err := tx.Commit().Error; err != nil {
		return TenantPurgeResult{TenantId: tenantID}, err
	}
	return result, nil
}
