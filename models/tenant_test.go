package models

import (
	"errors"

	"github.com/jinzhu/gorm"
	check "gopkg.in/check.v1"
)

func tenantTarget(email string) Target {
	return Target{BaseRecipient: BaseRecipient{Email: email, FirstName: "Tenant", LastName: "User"}}
}

func (s *ModelsSuite) createTenantGroup(c *check.C, tenantID, name string, emails ...string) Group {
	targets := make([]Target, 0, len(emails))
	for _, email := range emails {
		targets = append(targets, tenantTarget(email))
	}
	group := Group{Name: name, UserId: 1, TenantId: &tenantID, Targets: targets}
	c.Assert(PostGroup(&group), check.Equals, nil)
	return group
}

func (s *ModelsSuite) tenantCampaign(c *check.C, base Campaign, tenantID, name, groupName string) Campaign {
	campaign := base
	campaign.Name = name
	campaign.TenantId = &tenantID
	campaign.Groups = []Group{{Name: groupName}}
	c.Assert(PostCampaign(&campaign, 1), check.Equals, nil)
	return campaign
}

func tableCount(c *check.C, table string, query string, args ...interface{}) int64 {
	var count int64
	db.Table(table).Where(query, args...).Count(&count)
	return count
}

func (s *ModelsSuite) TestTenantGroupOwnershipAndTargetIsolation(c *check.C) {
	legacy := Group{Name: "Legacy", UserId: 1, Targets: []Target{tenantTarget("shared@example.com")}}
	c.Assert(PostGroup(&legacy), check.Equals, nil)
	c.Assert(legacy.TenantId, check.IsNil)
	c.Assert(legacy.Targets[0].TenantId, check.IsNil)

	tenantA := s.createTenantGroup(c, "tenant-a", "Tenant A", "shared@example.com")
	tenantB := s.createTenantGroup(c, "tenant-b", "Tenant B", "shared@example.com")
	c.Assert(tenantA.TenantId, check.NotNil)
	c.Assert(*tenantA.Targets[0].TenantId, check.Equals, "tenant-a")
	c.Assert(*tenantB.Targets[0].TenantId, check.Equals, "tenant-b")
	c.Assert(legacy.Targets[0].Id == tenantA.Targets[0].Id, check.Equals, false)
	c.Assert(tenantA.Targets[0].Id == tenantB.Targets[0].Id, check.Equals, false)
	c.Assert(tableCount(c, "targets", "email = ?", "shared@example.com"), check.Equals, int64(3))

	wrongTenant := "tenant-b"
	crossTenant := Group{
		Name:     "Invalid",
		UserId:   1,
		TenantId: tenantA.TenantId,
		Targets: []Target{{
			TenantId:      &wrongTenant,
			BaseRecipient: BaseRecipient{Email: "invalid@example.com"},
		}},
	}
	c.Assert(PostGroup(&crossTenant), check.Equals, ErrTenantMismatch)
}

func (s *ModelsSuite) TestTenantCampaignRejectsCrossTenantAndLegacyGroups(c *check.C) {
	base := s.createCampaignDependencies(c)
	s.createTenantGroup(c, "tenant-a", "Tenant A Group", "a@example.com")
	s.createTenantGroup(c, "tenant-b", "Tenant B Group", "b@example.com")

	tenantA := "tenant-a"
	campaign := base
	campaign.TenantId = &tenantA
	campaign.Groups = []Group{{Name: "Tenant B Group"}}
	c.Assert(PostCampaign(&campaign, 1), check.Equals, ErrGroupNotFound)

	campaign = base
	campaign.TenantId = &tenantA
	campaign.Groups = []Group{{Name: "Test Group"}}
	c.Assert(PostCampaign(&campaign, 1), check.Equals, ErrGroupNotFound)

	campaign = base
	campaign.Name = "Valid tenant campaign"
	campaign.TenantId = &tenantA
	campaign.Groups = []Group{{Name: "Tenant A Group"}}
	c.Assert(PostCampaign(&campaign, 1), check.Equals, nil)
	c.Assert(campaign.TenantId, check.NotNil)
}

func (s *ModelsSuite) TestTenantOwnershipCascadeConstraints(c *check.C) {
	base := s.createCampaignDependencies(c)
	c.Assert(PostCampaign(&base, 1), check.Equals, nil)
	c.Assert(tableCount(c, "results", "campaign_id = ?", base.Id), check.Equals, int64(4))
	c.Assert(tableCount(c, "events", "campaign_id = ?", base.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "mail_logs", "campaign_id = ?", base.Id), check.Equals, int64(4))
	c.Assert(db.Delete(&Campaign{Id: base.Id}).Error, check.Equals, nil)
	c.Assert(tableCount(c, "results", "campaign_id = ?", base.Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "events", "campaign_id = ?", base.Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "mail_logs", "campaign_id = ?", base.Id), check.Equals, int64(0))

	c.Assert(db.Exec("INSERT INTO results (campaign_id) VALUES (?)", int64(9223372036854775807)).Error, check.NotNil)
	c.Assert(db.Exec("INSERT INTO group_targets (group_id, target_id) VALUES (?, ?)", int64(9223372036854775807), int64(9223372036854775807)).Error, check.NotNil)

	tenantGroup := s.createTenantGroup(c, "cascade-tenant", "Cascade group", "cascade@example.com")
	c.Assert(db.Delete(&Tenant{Id: "cascade-tenant"}).Error, check.Equals, nil)
	c.Assert(tableCount(c, "groups", "id = ?", tenantGroup.Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "targets", "id = ?", tenantGroup.Targets[0].Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "group_targets", "group_id = ?", tenantGroup.Id), check.Equals, int64(0))
}

func (s *ModelsSuite) TestTenantPurgeIsolationIdempotencyAndRollback(c *check.C) {
	base := s.createCampaignDependencies(c)
	attachment := Attachment{TemplateId: base.Template.Id, Name: "shared.txt", Type: "text/plain", Content: "c2hhcmVk"}
	c.Assert(db.Save(&attachment).Error, check.Equals, nil)
	header := Header{SMTPId: base.SMTP.Id, Key: "X-Shared", Value: "true"}
	c.Assert(db.Save(&header).Error, check.Equals, nil)
	webhook := Webhook{Name: "Shared webhook", URL: "https://example.test/hook", IsActive: false}
	c.Assert(db.Save(&webhook).Error, check.Equals, nil)
	imap := IMAP{UserId: 1, Host: "imap.example.test", Port: 993, Username: "shared", Password: "shared"}
	c.Assert(db.Save(&imap).Error, check.Equals, nil)
	legacyCampaign := base
	legacyCampaign.Name = "Legacy campaign"
	c.Assert(PostCampaign(&legacyCampaign, 1), check.Equals, nil)

	tenantAGroup := s.createTenantGroup(c, "tenant-a", "Tenant A Group", "a1@example.com", "a2@example.com")
	tenantBGroup := s.createTenantGroup(c, "tenant-b", "Tenant B Group", "b@example.com")
	orphanedAfterFailure := s.createTenantGroup(c, "tenant-a", "Created before failed campaign", "a3@example.com")

	tenantACampaign := s.tenantCampaign(c, base, "tenant-a", "Tenant A campaign", tenantAGroup.Name)
	tenantBCampaign := s.tenantCampaign(c, base, "tenant-b", "Tenant B campaign", tenantBGroup.Name)

	tenantA := "tenant-a"
	tenantB := "tenant-b"
	c.Assert(PostEmailRequest(&EmailRequest{TenantId: &tenantA, UserId: 1, BaseRecipient: BaseRecipient{Email: "a-preview@example.com"}}), check.Equals, nil)
	c.Assert(PostEmailRequest(&EmailRequest{TenantId: &tenantB, UserId: 1, BaseRecipient: BaseRecipient{Email: "b-preview@example.com"}}), check.Equals, nil)
	legacyRequest := &EmailRequest{UserId: 1, BaseRecipient: BaseRecipient{Email: "legacy-preview@example.com"}}
	c.Assert(PostEmailRequest(legacyRequest), check.Equals, nil)

	failedCampaign := base
	failedCampaign.Name = "Failed cross-tenant campaign"
	failedCampaign.TenantId = &tenantA
	failedCampaign.Groups = []Group{{Name: tenantBGroup.Name}}
	c.Assert(PostCampaign(&failedCampaign, 1), check.Equals, ErrGroupNotFound)
	c.Assert(orphanedAfterFailure.Id > 0, check.Equals, true)

	rollbackErr := errors.New("forced purge failure")
	_, err := purgeTenant(tenantA, func(*gorm.DB) error { return rollbackErr })
	c.Assert(err, check.Equals, rollbackErr)
	c.Assert(tableCount(c, "campaigns", "tenant_id = ?", tenantA), check.Equals, int64(1))
	c.Assert(tableCount(c, "groups", "tenant_id = ?", tenantA), check.Equals, int64(2))
	c.Assert(tableCount(c, "results", "campaign_id = ?", tenantACampaign.Id), check.Equals, int64(2))

	result, err := PurgeTenant(tenantA)
	c.Assert(err, check.Equals, nil)
	c.Assert(result.TenantId, check.Equals, tenantA)
	c.Assert(result.Deleted.Campaigns, check.Equals, int64(1))
	c.Assert(result.Deleted.Results, check.Equals, int64(2))
	c.Assert(result.Deleted.Events > 0, check.Equals, true)
	c.Assert(result.Deleted.MailLogs, check.Equals, int64(2))
	c.Assert(result.Deleted.Groups, check.Equals, int64(2))
	c.Assert(result.Deleted.Targets, check.Equals, int64(3))
	c.Assert(result.Deleted.EmailRequests, check.Equals, int64(1))
	c.Assert(result.Deleted.Tenant, check.Equals, int64(1))

	for _, table := range []string{"campaigns", "groups", "targets", "email_requests"} {
		c.Assert(tableCount(c, table, "tenant_id = ?", tenantA), check.Equals, int64(0))
	}
	c.Assert(tableCount(c, "results", "campaign_id = ?", tenantACampaign.Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "events", "campaign_id = ?", tenantACampaign.Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "mail_logs", "campaign_id = ?", tenantACampaign.Id), check.Equals, int64(0))
	c.Assert(tableCount(c, "group_targets", "group_id = ?", tenantAGroup.Id), check.Equals, int64(0))

	c.Assert(tableCount(c, "campaigns", "id = ?", tenantBCampaign.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "results", "campaign_id = ?", tenantBCampaign.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "groups", "id = ?", tenantBGroup.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "email_requests", "tenant_id = ?", tenantB), check.Equals, int64(1))

	c.Assert(tableCount(c, "campaigns", "id = ? AND tenant_id IS NULL", legacyCampaign.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "groups", "id = ? AND tenant_id IS NULL", base.Groups[0].Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "targets", "tenant_id IS NULL"), check.Equals, int64(4))
	c.Assert(tableCount(c, "email_requests", "id = ? AND tenant_id IS NULL", legacyRequest.Id), check.Equals, int64(1))

	// Shared campaign dependencies are retained.
	c.Assert(tableCount(c, "templates", "id = ?", base.Template.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "pages", "id = ?", base.Page.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "smtp", "id = ?", base.SMTP.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "attachments", "id = ?", attachment.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "headers", "id = ?", header.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "webhooks", "id = ?", webhook.Id), check.Equals, int64(1))
	c.Assert(tableCount(c, "imap", "user_id = ?", int64(1)), check.Equals, int64(1))
	c.Assert(tableCount(c, "users", "id = ?", int64(1)), check.Equals, int64(1))
	c.Assert(tableCount(c, "roles", "1 = 1"), check.Equals, int64(2))
	c.Assert(tableCount(c, "permissions", "1 = 1") > 0, check.Equals, true)
	c.Assert(tableCount(c, "role_permissions", "1 = 1") > 0, check.Equals, true)

	repeated, err := PurgeTenant(tenantA)
	c.Assert(err, check.Equals, nil)
	c.Assert(repeated.Deleted, check.DeepEquals, TenantDeletionCounts{})
	unknown, err := PurgeTenant("unknown-tenant")
	c.Assert(err, check.Equals, nil)
	c.Assert(unknown.Deleted, check.DeepEquals, TenantDeletionCounts{})
}
