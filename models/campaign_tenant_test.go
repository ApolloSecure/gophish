package models

import (
	"fmt"
	"sync/atomic"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	check "gopkg.in/check.v1"
)

type queryCounter struct {
	count int64
}

func (q *queryCounter) Print(values ...interface{}) {
	if len(values) > 0 && values[0] == "sql" {
		atomic.AddInt64(&q.count, 1)
	}
}

func (s *ModelsSuite) TestCampaignTenantIDValidationAndPersistence(c *check.C) {
	tenantID := "lQJ-D6j-wvw"
	campaign := s.createCampaignDependencies(c)
	campaign.TenantId = &tenantID
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.Equals, nil)

	got, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(got.TenantId, check.NotNil)
	c.Assert(*got.TenantId, check.Equals, tenantID)

	withoutTenant := s.createCampaignDependencies(c)
	withoutTenant.Name = "Campaign without tenant"
	c.Assert(PostCampaign(&withoutTenant, withoutTenant.UserId), check.Equals, nil)
	got, err = GetCampaign(withoutTenant.Id, withoutTenant.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(got.TenantId, check.IsNil)

	for _, invalid := range []string{"", " surrounded ", string(make([]byte, 256))} {
		campaign := s.createCampaignDependencies(c)
		campaign.TenantId = &invalid
		c.Assert(PostCampaign(&campaign, campaign.UserId), check.Equals, ErrInvalidTenantID)
	}
}

func (s *ModelsSuite) TestTenantFilteredCampaignsAndBatchResults(c *check.C) {
	tenantA := "tenant-a"
	tenantB := "tenant-b"
	dependencies := s.createCampaignDependencies(c)

	campaignA1 := dependencies
	campaignA1.Name = "Tenant A campaign 1"
	campaignA1.TenantId = &tenantA
	c.Assert(PostCampaign(&campaignA1, 1), check.Equals, nil)
	campaignA2 := dependencies
	campaignA2.Name = "Tenant A campaign 2"
	campaignA2.TenantId = &tenantA
	c.Assert(PostCampaign(&campaignA2, 1), check.Equals, nil)
	campaignB := dependencies
	campaignB.Name = "Tenant B campaign"
	campaignB.TenantId = &tenantB
	c.Assert(PostCampaign(&campaignB, 1), check.Equals, nil)
	legacy := dependencies
	legacy.Name = "Legacy campaign"
	c.Assert(PostCampaign(&legacy, 1), check.Equals, nil)

	filtered, err := GetCampaignsByTenant(1, tenantA, true)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(filtered), check.Equals, 2)
	c.Assert(filtered[0].Id, check.Equals, campaignA2.Id)
	c.Assert(filtered[1].Id, check.Equals, campaignA1.Id)
	for _, campaign := range filtered {
		c.Assert(campaign.TenantId, check.NotNil)
		c.Assert(*campaign.TenantId, check.Equals, tenantA)
		c.Assert(len(campaign.Results), check.Equals, 4)
		c.Assert(campaign.Summary, check.NotNil)
		c.Assert(campaign.Summary.Total, check.Equals, int64(4))
	}
	firstPage, err := GetCampaignsByTenantPage(1, tenantA, false, 1, 1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(firstPage), check.Equals, 1)
	c.Assert(firstPage[0].Id, check.Equals, campaignA2.Id)
	secondPage, err := GetCampaignsByTenantPage(1, tenantA, false, 2, 1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(secondPage), check.Equals, 1)
	c.Assert(secondPage[0].Id, check.Equals, campaignA1.Id)

	unknown, err := GetCampaignsByTenant(1, "unknown", true)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(unknown), check.Equals, 0)
	otherUser, err := GetCampaignsByTenant(2, tenantA, true)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(otherUser), check.Equals, 0)

	_, err = GetCampaignResultsForTenant(campaignB.Id, 1, tenantA)
	c.Assert(err, check.Equals, gorm.ErrRecordNotFound)
	results, err := GetCampaignResultsForTenant(campaignA1.Id, 1, tenantA)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(results.Results), check.Equals, len(filtered[0].Results))
	c.Assert(results.Summary.Total, check.Equals, filtered[0].Summary.Total)

	unfiltered, err := GetCampaigns(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(unfiltered), check.Equals, 4)
}

func (s *ModelsSuite) TestTenantBatchQueryCountIsBounded(c *check.C) {
	tenantID := "query-count-tenant"
	dependencies := s.createCampaignDependencies(c)
	for i := 0; i < 10; i++ {
		campaign := dependencies
		campaign.Name = fmt.Sprintf("Tenant campaign %d", i)
		campaign.TenantId = &tenantID
		c.Assert(PostCampaign(&campaign, 1), check.Equals, nil)
	}

	counter := &queryCounter{}
	db.SetLogger(counter)
	db.LogMode(true)
	_, err := GetCampaignsByTenant(1, tenantID, true)
	db.LogMode(false)
	db.SetLogger(log.Logger)
	c.Assert(err, check.Equals, nil)
	c.Assert(atomic.LoadInt64(&counter.count), check.Equals, int64(8))
}
