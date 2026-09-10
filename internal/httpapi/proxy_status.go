package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/telemt"
)

func NewWithProxyHealth(db *sql.DB, options Options, checker telemt.Checker) *Server {
	s := New(db, options)
	s.mux.HandleFunc("GET /api/system/proxy", s.handleProxyStatus(checker))
	return s
}

func (s *Server) handleProxyStatus(checker telemt.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, ok, err := s.currentSession(r)
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if !ok {
			writeProblem(w, Problem{
				Type:     "about:blank",
				Title:    "Unauthorized",
				Status:   http.StatusUnauthorized,
				Code:     "AUTH_REQUIRED",
				Message:  "Administrator authentication is required.",
				Instance: r.URL.Path,
			})
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		if checker == nil {
			writeJSON(w, http.StatusOK, telemt.Health{State: telemt.StateNotConfigured})
			return
		}
		writeJSON(w, http.StatusOK, checker.Health(r.Context()))
	}
}
