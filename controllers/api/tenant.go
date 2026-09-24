package api

import (
	"net/http"

	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// Tenant handles idempotent deletion of directly and indirectly tenant-owned data.
func (as *Server) Tenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	tenantID := mux.Vars(r)["tenantId"]
	if err := models.ValidateTenantID(&tenantID); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	result, err := models.PurgeTenant(tenantID)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error deleting tenant data"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, result, http.StatusOK)
}
