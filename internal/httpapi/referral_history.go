package httpapi

import (
	"net/http"
	"strconv"

	"github.com/smorad3363/teleproxy/internal/referral"
)

func (s *Server) registerReferralHistoryRoutes() {
	s.mux.HandleFunc("GET /api/referral/history", s.handleReferralHistoryList)
}

func (s *Server) handleReferralHistoryList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	query, ok := parseReferralHistoryQuery(w, r)
	if !ok {
		return
	}
	page, err := referral.History(r.Context(), s.db, query)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"history":        page.Items,
		"next_before_id": page.NextBeforeID,
	})
}

func parseReferralHistoryQuery(w http.ResponseWriter, r *http.Request) (referral.HistoryQuery, bool) {
	values := r.URL.Query()
	for key := range values {
		if key != "before_id" && key != "limit" && key != "status" {
			writeReferralHistoryProblem(w, r, "The referral history query is invalid.")
			return referral.HistoryQuery{}, false
		}
	}

	var query referral.HistoryQuery
	if raw, exists := values["before_id"]; exists {
		if len(raw) != 1 {
			writeReferralHistoryProblem(w, r, "The referral history cursor is invalid.")
			return referral.HistoryQuery{}, false
		}
		value, err := strconv.ParseInt(raw[0], 10, 64)
		if err != nil || value <= 0 {
			writeReferralHistoryProblem(w, r, "The referral history cursor is invalid.")
			return referral.HistoryQuery{}, false
		}
		query.BeforeID = value
	}
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			writeReferralHistoryProblem(w, r, "The referral history limit is invalid.")
			return referral.HistoryQuery{}, false
		}
		value, err := strconv.Atoi(raw[0])
		if err != nil || value < 1 || value > referral.MaxHistoryLimit {
			writeReferralHistoryProblem(w, r, "The referral history limit is invalid.")
			return referral.HistoryQuery{}, false
		}
		query.Limit = value
	}
	if raw, exists := values["status"]; exists {
		if len(raw) != 1 {
			writeReferralHistoryProblem(w, r, "The referral history status is invalid.")
			return referral.HistoryQuery{}, false
		}
		status := referral.Status(raw[0])
		if status != referral.StatusPending && status != referral.StatusRewarded && status != referral.StatusRejected {
			writeReferralHistoryProblem(w, r, "The referral history status is invalid.")
			return referral.HistoryQuery{}, false
		}
		query.Status = status
	}
	return query, true
}

func writeReferralHistoryProblem(w http.ResponseWriter, r *http.Request, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "REFERRAL_HISTORY_INVALID", Message: message, Instance: r.URL.Path})
}
