package httpapi

import (
	"net/http"

	"github.com/smorad3363/teleproxy/internal/admin"
)

func (s *Server) registerAdminInventoryRoutes() {
	s.mux.HandleFunc("GET /admins", s.handleAdminInventoryPage)
	s.mux.HandleFunc("GET /api/admins", s.handleAdminInventoryList)
}

func (s *Server) handleAdminInventoryList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	if !validateAdminInventoryQuery(w, r) {
		return
	}
	entries, err := admin.ListInventory(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"admins": entries})
}

func validateAdminInventoryQuery(w http.ResponseWriter, r *http.Request) bool {
	if len(r.URL.Query()) == 0 {
		return true
	}
	writeProblem(w, Problem{
		Type:     "about:blank",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Code:     "ADMIN_INVENTORY_INVALID",
		Message:  "The administrator inventory query is invalid.",
		Instance: r.URL.Path,
	})
	return false
}
