package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/useradmin"
)

func (s *Server) registerUserAdminRoutes() {
	s.mux.HandleFunc("GET /users", s.handleUserAdminPage)
	s.mux.HandleFunc("GET /api/users", s.handleUserAdminList)
	s.registerAuditLogRoutes()
	s.registerAdminInventoryRoutes()
}

func (s *Server) handleUserAdminList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	query, ok := parseUserAdminQuery(w, r)
	if !ok {
		return
	}
	page, err := useradmin.List(r.Context(), s.db, query, time.Now().UTC())
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"users":          page.Items,
		"next_before_id": page.NextBeforeID,
	})
}

func parseUserAdminQuery(w http.ResponseWriter, r *http.Request) (useradmin.ListQuery, bool) {
	values := r.URL.Query()
	for key := range values {
		if key != "before_id" && key != "limit" && key != "telegram_id" && key != "proxy_username" && key != "provisioning_phase" {
			writeUserAdminProblem(w, r, "The user inventory query is invalid.")
			return useradmin.ListQuery{}, false
		}
	}

	var query useradmin.ListQuery
	if raw, exists := values["before_id"]; exists {
		if len(raw) != 1 {
			writeUserAdminProblem(w, r, "The user inventory cursor is invalid.")
			return useradmin.ListQuery{}, false
		}
		value, err := strconv.ParseInt(raw[0], 10, 64)
		if err != nil || value <= 0 {
			writeUserAdminProblem(w, r, "The user inventory cursor is invalid.")
			return useradmin.ListQuery{}, false
		}
		query.BeforeID = value
	}
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			writeUserAdminProblem(w, r, "The user inventory limit is invalid.")
			return useradmin.ListQuery{}, false
		}
		value, err := strconv.Atoi(raw[0])
		if err != nil || value < 1 || value > useradmin.MaxListLimit {
			writeUserAdminProblem(w, r, "The user inventory limit is invalid.")
			return useradmin.ListQuery{}, false
		}
		query.Limit = value
	}
	if raw, exists := values["telegram_id"]; exists {
		if len(raw) != 1 {
			writeUserAdminProblem(w, r, "The user inventory Telegram ID is invalid.")
			return useradmin.ListQuery{}, false
		}
		value, err := strconv.ParseInt(raw[0], 10, 64)
		if err != nil || value <= 0 {
			writeUserAdminProblem(w, r, "The user inventory Telegram ID is invalid.")
			return useradmin.ListQuery{}, false
		}
		query.TelegramID = value
	}
	if raw, exists := values["proxy_username"]; exists {
		if len(raw) != 1 || proxyuser.ValidateUsername(raw[0]) != nil {
			writeUserAdminProblem(w, r, "The user inventory proxy username is invalid.")
			return useradmin.ListQuery{}, false
		}
		query.ProxyUsername = raw[0]
	}
	if raw, exists := values["provisioning_phase"]; exists {
		if len(raw) != 1 {
			writeUserAdminProblem(w, r, "The user inventory provisioning phase is invalid.")
			return useradmin.ListQuery{}, false
		}
		phase := proxyprovision.Phase(raw[0])
		switch phase {
		case proxyprovision.PhasePrepared, proxyprovision.PhaseOwned, proxyprovision.PhaseCollision:
			query.ProvisioningPhase = phase
		default:
			writeUserAdminProblem(w, r, "The user inventory provisioning phase is invalid.")
			return useradmin.ListQuery{}, false
		}
	}
	return query, true
}

func writeUserAdminProblem(w http.ResponseWriter, r *http.Request, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "USER_INVENTORY_INVALID", Message: message, Instance: r.URL.Path})
}
