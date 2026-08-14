package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

func tenantCampaignPayload(name, tenantID string) []byte {
	return []byte(fmt.Sprintf(`{
		"tenant_id": %q,
		"name": %q,
		"groups": [{"name": "Test Group"}],
		"template": {"name": "Test Template"},
		"page": {"name": "Test Page"},
		"smtp": {"name": "Test Page"},
		"url": "https://example.test",
		"launch_date": %q
	}`, tenantID, name, time.Now().UTC().Add(time.Hour).Format(time.RFC3339)))
}

func campaignRequest(method, target string, body []byte) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	return ctx.Set(req, "user_id", int64(1))
}

func TestTenantCampaignCreateListAndBatchResults(t *testing.T) {
	testContext := setupTest(t)
	createTestData(t)

	created := make(map[string]models.Campaign)
	for _, tenantID := range []string{"tenant-a", "tenant-b"} {
		recorder := httptest.NewRecorder()
		testContext.apiServer.Campaigns(recorder, campaignRequest(http.MethodPost, "/api/campaigns/", tenantCampaignPayload("Campaign "+tenantID, tenantID)))
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create campaign: status %d body %s", recorder.Code, recorder.Body.String())
		}
		var campaign models.Campaign
		if err := json.NewDecoder(recorder.Body).Decode(&campaign); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		if campaign.TenantId == nil || *campaign.TenantId != tenantID {
			t.Fatalf("unexpected tenant_id in create response: %#v", campaign.TenantId)
		}
		created[tenantID] = campaign
	}

	recorder := httptest.NewRecorder()
	testContext.apiServer.Campaigns(recorder, campaignRequest(http.MethodGet, "/api/campaigns/?tenant_id=tenant-a&include_results=true", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list tenant campaigns: status %d body %s", recorder.Code, recorder.Body.String())
	}
	var campaigns []models.Campaign
	if err := json.NewDecoder(recorder.Body).Decode(&campaigns); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(campaigns) != 1 || campaigns[0].TenantId == nil || *campaigns[0].TenantId != "tenant-a" {
		t.Fatalf("tenant list leaked or omitted campaigns: %#v", campaigns)
	}
	if len(campaigns[0].Results) != 2 || campaigns[0].Summary == nil || campaigns[0].Summary.Total != 2 {
		t.Fatalf("batch results were incomplete: %#v", campaigns[0])
	}

	resultRecorder := httptest.NewRecorder()
	resultRequest := campaignRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/results?tenant_id=tenant-a", created["tenant-a"].Id), nil)
	resultRequest = mux.SetURLVars(resultRequest, map[string]string{"id": fmt.Sprint(created["tenant-a"].Id)})
	testContext.apiServer.CampaignResults(resultRecorder, resultRequest)
	if resultRecorder.Code != http.StatusOK {
		t.Fatalf("get individual results: status %d body %s", resultRecorder.Code, resultRecorder.Body.String())
	}
	var individual models.CampaignResults
	if err := json.NewDecoder(resultRecorder.Body).Decode(&individual); err != nil {
		t.Fatalf("decode individual results: %v", err)
	}
	if len(individual.Results) != len(campaigns[0].Results) || individual.Summary.Total != campaigns[0].Summary.Total {
		t.Fatalf("batch and individual results differ: batch=%#v individual=%#v", campaigns[0], individual)
	}

	crossTenantRecorder := httptest.NewRecorder()
	crossTenantRequest := campaignRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%d/results?tenant_id=tenant-a", created["tenant-b"].Id), nil)
	crossTenantRequest = mux.SetURLVars(crossTenantRequest, map[string]string{"id": fmt.Sprint(created["tenant-b"].Id)})
	testContext.apiServer.CampaignResults(crossTenantRecorder, crossTenantRequest)
	if crossTenantRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant results status = %d, want 404", crossTenantRecorder.Code)
	}

	unknownRecorder := httptest.NewRecorder()
	testContext.apiServer.Campaigns(unknownRecorder, campaignRequest(http.MethodGet, "/api/campaigns/?tenant_id=unknown&include_results=true", nil))
	var unknown []models.Campaign
	decodeErr := json.NewDecoder(unknownRecorder.Body).Decode(&unknown)
	if unknownRecorder.Code != http.StatusOK || decodeErr != nil || len(unknown) != 0 {
		t.Fatalf("unknown tenant response: status %d body %s", unknownRecorder.Code, unknownRecorder.Body.String())
	}
}

func TestTenantCampaignQueryValidation(t *testing.T) {
	testContext := setupTest(t)
	for _, target := range []string{
		"/api/campaigns/?tenant_id=",
		"/api/campaigns/?tenant_id=%20tenant%20",
		"/api/campaigns/?tenant_id=tenant-a&include_results=maybe",
		"/api/campaigns/?tenant_id=tenant-a&page=0",
		"/api/campaigns/?tenant_id=tenant-a&page=not-a-number",
		"/api/campaigns/?tenant_id=tenant-a&limit=101",
		"/api/campaigns/?include_results=true",
	} {
		recorder := httptest.NewRecorder()
		testContext.apiServer.Campaigns(recorder, campaignRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want 400", target, recorder.Code)
		}
	}
}

func TestTenantCampaignPaginationAndOrdering(t *testing.T) {
	testContext := setupTest(t)
	createTestData(t)
	createdIDs := make([]int64, 0, 3)
	for i := 0; i < 3; i++ {
		recorder := httptest.NewRecorder()
		testContext.apiServer.Campaigns(recorder, campaignRequest(http.MethodPost, "/api/campaigns/", tenantCampaignPayload(fmt.Sprintf("Paged campaign %d", i), "paged-tenant")))
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create campaign %d: status %d body %s", i, recorder.Code, recorder.Body.String())
		}
		var campaign models.Campaign
		if err := json.NewDecoder(recorder.Body).Decode(&campaign); err != nil {
			t.Fatalf("decode campaign %d: %v", i, err)
		}
		createdIDs = append(createdIDs, campaign.Id)
	}

	for page, expectedID := range []int64{createdIDs[2], createdIDs[1], createdIDs[0]} {
		recorder := httptest.NewRecorder()
		target := fmt.Sprintf("/api/campaigns/?tenant_id=paged-tenant&page=%d&limit=1", page+1)
		testContext.apiServer.Campaigns(recorder, campaignRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("page %d: status %d body %s", page+1, recorder.Code, recorder.Body.String())
		}
		var campaigns []models.Campaign
		if err := json.NewDecoder(recorder.Body).Decode(&campaigns); err != nil {
			t.Fatalf("decode page %d: %v", page+1, err)
		}
		if len(campaigns) != 1 || campaigns[0].Id != expectedID {
			t.Fatalf("page %d campaigns = %#v, want campaign %d", page+1, campaigns, expectedID)
		}
	}
}
