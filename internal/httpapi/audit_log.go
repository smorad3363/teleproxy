package httpapi

import (
	"net/http"
	"strconv"

	"github.com/smorad3363/teleproxy/internal/auditlog"
)

const (
	defaultAuditLogLimit = 50
	maxAuditLogLimit     = 100
)

func (s *Server) registerAuditLogRoutes() {
	s.mux.HandleFunc("GET /api/audit-log", s.handleAuditLogList)
}

func (s *Server) handleAuditLogList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	beforeID, limit, ok := parseAuditLogQuery(w, r)
	if !ok {
		return
	}
	entries, err := auditlog.List(r.Context(), s.db, beforeID, limit)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	var nextBeforeID *int64
	if len(entries) == limit {
		lastID := entries[len(entries)-1].ID
		more, err := auditlog.List(r.Context(), s.db, lastID, 1)
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if len(more) > 0 {
			nextBeforeID = &lastID
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"entries":        entries,
		"next_before_id": nextBeforeID,
	})
}

func parseAuditLogQuery(w http.ResponseWriter, r *http.Request) (int64, int, bool) {
	values := r.URL.Query()
	for key := range values {
		if key != "before_id" && key != "limit" {
			writeAuditLogProblem(w, r, "The audit log query is invalid.")
			return 0, 0, false
		}
	}

	var beforeID int64
	if raw, exists := values["before_id"]; exists {
		if len(raw) != 1 {
			writeAuditLogProblem(w, r, "The audit log cursor is invalid.")
			return 0, 0, false
		}
		value, err := strconv.ParseInt(raw[0], 10, 64)
		if err != nil || value <= 0 {
			writeAuditLogProblem(w, r, "The audit log cursor is invalid.")
			return 0, 0, false
		}
		beforeID = value
	}

	limit := defaultAuditLogLimit
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			writeAuditLogProblem(w, r, "The audit log limit is invalid.")
			return 0, 0, false
		}
		value, err := strconv.Atoi(raw[0])
		if err != nil || value < 1 || value > maxAuditLogLimit {
			writeAuditLogProblem(w, r, "The audit log limit is invalid.")
			return 0, 0, false
		}
		limit = value
	}
	return beforeID, limit, true
}

func writeAuditLogProblem(w http.ResponseWriter, r *http.Request, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "AUDIT_LOG_INVALID", Message: message, Instance: r.URL.Path})
}
