package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophish/gophish/models"
)

func TestGroupCustomFieldsAPIRoundTrip(t *testing.T) {
	ctx := setupTest(t)
	body := []byte(`{
		"name": "API Custom Fields",
		"targets": [{
			"email": "target@example.com",
			"custom_fields": {
				"Department": "Finance",
				"AccountName": "Example Corp"
			}
		}]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups/", bytes.NewReader(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.apiKey))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	ctx.apiServer.ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("unexpected create status %d: %s", response.Code, response.Body.String())
	}

	created := models.Group{}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("error decoding create response: %v", err)
	}
	if got := created.Targets[0].CustomFields["Department"]; got != "Finance" {
		t.Fatalf("unexpected custom field in create response: %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/groups/%d", created.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.apiKey))
	response = httptest.NewRecorder()
	ctx.apiServer.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected get status %d: %s", response.Code, response.Body.String())
	}

	got := models.Group{}
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("error decoding get response: %v", err)
	}
	if got.Targets[0].CustomFields["AccountName"] != "Example Corp" {
		t.Fatalf("custom fields did not round trip through the API: %#v", got.Targets[0].CustomFields)
	}
}
