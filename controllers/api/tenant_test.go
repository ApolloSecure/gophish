package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

func TestTenantPurgeEndpointAuthenticationContractAndIdempotency(t *testing.T) {
	ctx := setupTest(t)
	tenantID := "api-tenant"
	group := models.Group{
		Name:     "API tenant group",
		UserId:   1,
		TenantId: &tenantID,
		Targets: []models.Target{{
			BaseRecipient: models.BaseRecipient{Email: "api-tenant@example.com"},
		}},
	}
	if err := models.PostGroup(&group); err != nil {
		t.Fatalf("create tenant group: %v", err)
	}

	unauthorized := httptest.NewRecorder()
	ctx.apiServer.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodDelete, "/api/tenants/"+tenantID, nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorized.Code)
	}

	request := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/tenants/"+id, nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.apiKey))
		response := httptest.NewRecorder()
		ctx.apiServer.ServeHTTP(response, req)
		return response
	}

	response := request(tenantID)
	if response.Code != http.StatusOK {
		t.Fatalf("purge status = %d: %s", response.Code, response.Body.String())
	}
	result := models.TenantPurgeResult{}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode purge response: %v", err)
	}
	if result.TenantId != tenantID || result.Deleted.Groups != 1 || result.Deleted.Targets != 1 || result.Deleted.Tenant != 1 {
		t.Fatalf("unexpected purge response: %#v", result)
	}

	repeated := request(tenantID)
	if repeated.Code != http.StatusOK {
		t.Fatalf("repeat purge status = %d: %s", repeated.Code, repeated.Body.String())
	}
	repeatedResult := models.TenantPurgeResult{}
	if err := json.NewDecoder(repeated.Body).Decode(&repeatedResult); err != nil {
		t.Fatalf("decode repeated purge response: %v", err)
	}
	if repeatedResult.Deleted != (models.TenantDeletionCounts{}) {
		t.Fatalf("repeat purge deleted records: %#v", repeatedResult.Deleted)
	}
}

func TestTenantPurgeEndpointRejectsInvalidTenantID(t *testing.T) {
	ctx := setupTest(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/tenants/%20invalid%20", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.apiKey))
	response := httptest.NewRecorder()
	ctx.apiServer.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid tenant status = %d, want 400: %s", response.Code, response.Body.String())
	}

	emptyRequest := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/api/tenants/", nil), map[string]string{"tenantId": ""})
	emptyResponse := httptest.NewRecorder()
	ctx.apiServer.Tenant(emptyResponse, emptyRequest)
	if emptyResponse.Code != http.StatusBadRequest {
		t.Fatalf("empty tenant status = %d, want 400: %s", emptyResponse.Code, emptyResponse.Body.String())
	}
}
