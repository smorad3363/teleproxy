package httpapi

import (
	"errors"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

const quotaReconcileUnavailableCode = "QUOTA_RECONCILE_UNAVAILABLE"

type quotaReconcileTrigger interface {
	Trigger(string) error
}

func (s *Server) registerQuotaReconcileRoutes(trigger quotaReconcileTrigger) {
	s.mux.HandleFunc("POST /api/proxy/users/{username}/reconcile", s.handleProxyUserReconcile(trigger))
}

func (s *Server) handleProxyUserReconcile(trigger quotaReconcileTrigger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requireAdminAPI(w, r, true) {
			return
		}
		username := r.PathValue("username")
		if err := proxyuser.ValidateUsername(username); err != nil {
			writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "PROXY_USER_INVALID", Message: "The proxy username is invalid.", Instance: r.URL.Path})
			return
		}
		if _, err := proxyuser.Get(r.Context(), s.db, username); errors.Is(err, proxyuser.ErrNotFound) {
			writeProblem(w, Problem{Type: "about:blank", Title: "Not Found", Status: http.StatusNotFound, Code: "PROXY_USER_NOT_FOUND", Message: "The proxy user was not found.", Instance: r.URL.Path})
			return
		} else if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if trigger == nil {
			writeProblem(w, Problem{Type: "about:blank", Title: "Service Unavailable", Status: http.StatusServiceUnavailable, Code: "QUOTA_RECONCILE_NOT_CONFIGURED", Message: "Quota reconciliation is not configured.", Instance: r.URL.Path})
			return
		}
		if err := trigger.Trigger(username); err != nil {
			if _, markErr := proxyuser.MarkSyncError(r.Context(), s.db, username, quotaReconcileUnavailableCode); markErr != nil {
				s.writeInternalError(w, r)
				return
			}
			writeProblem(w, Problem{Type: "about:blank", Title: "Service Unavailable", Status: http.StatusServiceUnavailable, Code: quotaReconcileUnavailableCode, Message: "Quota reconciliation could not be queued.", Instance: r.URL.Path})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued", "username": username})
	}
}
