package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

// Campaigns returns a list of campaigns if requested via GET.
// If requested via POST, APICampaigns creates a new campaign and returns a reference to it.
func (as *Server) Campaigns(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		tenantID, tenantSupplied, err := tenantIDFromRequest(r)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		includeResults := false
		if value := r.URL.Query().Get("include_results"); value != "" {
			includeResults, err = strconv.ParseBool(value)
			if err != nil {
				JSONResponse(w, models.Response{Success: false, Message: "include_results must be true or false"}, http.StatusBadRequest)
				return
			}
		}
		var cs []models.Campaign
		if tenantSupplied {
			page, limit, paginationErr := campaignPaginationFromRequest(r)
			if paginationErr != nil {
				JSONResponse(w, models.Response{Success: false, Message: paginationErr.Error()}, http.StatusBadRequest)
				return
			}
			cs, err = models.GetCampaignsByTenantPage(ctx.Get(r, "user_id").(int64), tenantID, includeResults, page, limit)
		} else {
			if includeResults {
				JSONResponse(w, models.Response{Success: false, Message: "tenant_id is required when include_results is supplied"}, http.StatusBadRequest)
				return
			}
			cs, err = models.GetCampaigns(ctx.Get(r, "user_id").(int64))
		}
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error retrieving campaigns"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cs, http.StatusOK)
	//POST: Create a new campaign and return it as JSON
	case r.Method == "POST":
		c := models.Campaign{}
		// Put the request into a campaign
		err := json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}
		err = models.PostCampaign(&c, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		// If the campaign is scheduled to launch immediately, send it to the worker.
		// Otherwise, the worker will pick it up at the scheduled time
		if c.Status == models.CampaignInProgress {
			go as.worker.LaunchCampaign(c)
		}
		JSONResponse(w, c, http.StatusCreated)
	}
}

// CampaignsSummary returns the summary for the current user's campaigns
func (as *Server) CampaignsSummary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		cs, err := models.GetCampaignSummaries(ctx.Get(r, "user_id").(int64))
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cs, http.StatusOK)
	}
}

// Campaign returns details about the requested campaign. If the campaign is not
// valid, APICampaign returns null.
func (as *Server) Campaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	tenantID, tenantSupplied, err := tenantIDFromRequest(r)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	var c models.Campaign
	if tenantSupplied {
		c, err = models.GetCampaignForTenant(id, ctx.Get(r, "user_id").(int64), tenantID)
	} else {
		c, err = models.GetCampaign(id, ctx.Get(r, "user_id").(int64))
	}
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, c, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteCampaign(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting campaign"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign deleted successfully!"}, http.StatusOK)
	}
}

// CampaignResults returns just the results for a given campaign to
// significantly reduce the information returned.
func (as *Server) CampaignResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	tenantID, tenantSupplied, err := tenantIDFromRequest(r)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	var cr models.CampaignResults
	if tenantSupplied {
		cr, err = models.GetCampaignResultsForTenant(id, ctx.Get(r, "user_id").(int64), tenantID)
	} else {
		cr, err = models.GetCampaignResults(id, ctx.Get(r, "user_id").(int64))
	}
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}
	if r.Method == "GET" {
		JSONResponse(w, cr, http.StatusOK)
		return
	}
}

func tenantIDFromRequest(r *http.Request) (string, bool, error) {
	values, supplied := r.URL.Query()["tenant_id"]
	if !supplied {
		return "", false, nil
	}
	tenantID := ""
	if len(values) > 0 {
		tenantID = values[0]
	}
	if err := models.ValidateTenantID(&tenantID); err != nil {
		return "", true, err
	}
	return tenantID, true, nil
}

func campaignPaginationFromRequest(r *http.Request) (int, int, error) {
	page, err := positiveQueryInt(r, "page", 1)
	if err != nil {
		return 0, 0, err
	}
	limit, err := positiveQueryInt(r, "limit", 50)
	if err != nil {
		return 0, 0, err
	}
	if limit > 100 {
		return 0, 0, errors.New("limit must be no greater than 100")
	}
	if page > int(^uint(0)>>1)/limit {
		return 0, 0, errors.New("page is too large")
	}
	return page, limit, nil
}

func positiveQueryInt(r *http.Request, name string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

// CampaignSummary returns the summary for a given campaign.
func (as *Server) CampaignSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch {
	case r.Method == "GET":
		cs, err := models.GetCampaignSummary(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
			} else {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			}
			log.Error(err)
			return
		}
		JSONResponse(w, cs, http.StatusOK)
	}
}

// CampaignComplete effectively "ends" a campaign.
// Future phishing emails clicked will return a simple "404" page.
func (as *Server) CampaignComplete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch {
	case r.Method == "GET":
		err := models.CompleteCampaign(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error completing campaign"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign completed successfully!"}, http.StatusOK)
	}
}
