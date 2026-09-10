package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

const maxProxyUserBodyBytes = 16 << 10

type proxyUserLifecycle interface {
	CreateUser(context.Context, string, bool) (telemt.Credential, error)
	SetUserEnabled(context.Context, string, bool) (telemt.User, error)
	RotateUserSecret(context.Context, string) (telemt.Credential, error)
}

func NewWithProxyClient(db *sql.DB, options Options, client *telemt.Client) *Server {
	return NewWithProxyServices(db, options, client, nil)
}

func NewWithProxyServices(db *sql.DB, options Options, client *telemt.Client, trigger quotaReconcileTrigger) *Server {
	if client == nil {
		return newWithProxyDependenciesAndReconciler(db, options, nil, nil, trigger)
	}
	return newWithProxyDependenciesAndReconciler(db, options, client, client, trigger)
}

func newWithProxyDependencies(db *sql.DB, options Options, checker telemt.Checker, lifecycle proxyUserLifecycle) *Server {
	return newWithProxyDependenciesAndReconciler(db, options, checker, lifecycle, nil)
}

func newWithProxyDependenciesAndReconciler(db *sql.DB, options Options, checker telemt.Checker, lifecycle proxyUserLifecycle, trigger quotaReconcileTrigger) *Server {
	s := NewWithProxyHealth(db, options, checker)
	s.registerProxyUserRoutes(lifecycle, trigger)
	s.registerQuotaReconcileRoutes(trigger)
	return s
}

func (s *Server) registerProxyUserRoutes(lifecycle proxyUserLifecycle, trigger quotaReconcileTrigger) {
	s.mux.HandleFunc("GET /api/proxy/users", s.handleProxyUsersList)
	s.mux.HandleFunc("POST /api/proxy/users", s.handleProxyUserCreate(lifecycle, trigger))
	s.mux.HandleFunc("POST /api/proxy/users/{username}/enable", s.handleProxyUserEnabled(lifecycle, trigger, true))
	s.mux.HandleFunc("POST /api/proxy/users/{username}/disable", s.handleProxyUserEnabled(lifecycle, trigger, false))
	s.mux.HandleFunc("POST /api/proxy/users/{username}/rotate-secret", s.handleProxyUserRotate(lifecycle))
}

func (s *Server) handleProxyUsersList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	users, err := proxyuser.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (s *Server) handleProxyUserCreate(lifecycle proxyUserLifecycle, trigger quotaReconcileTrigger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requireAdminAPI(w, r, true) {
			return
		}
		var input struct {
			Username string `json:"username"`
			Enabled  *bool  `json:"enabled"`
		}
		if !decodeProxyJSON(w, r, &input) {
			return
		}
		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		input.Username = strings.TrimSpace(input.Username)
		if err := proxyuser.ValidateUsername(input.Username); err != nil {
			writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "PROXY_USER_INVALID", Message: "The proxy user request is invalid.", Instance: r.URL.Path})
			return
		}
		created, err := proxyuser.Create(r.Context(), s.db, input.Username, enabled)
		if errors.Is(err, proxyuser.ErrExists) {
			writeProblem(w, Problem{Type: "about:blank", Title: "Conflict", Status: http.StatusConflict, Code: "PROXY_USER_EXISTS", Message: "The proxy user already exists.", Instance: r.URL.Path})
			return
		}
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if lifecycle == nil {
			s.markProxySyncFailure(w, r, created.Username, "TELEMT_NOT_CONFIGURED", http.StatusServiceUnavaile)
			return
		}
		dataPlaneEnabled := created.DesiredEnable
		if trigger != nil {
			// New users stay fail-closed until quota bootstrap has been applied.
			dataPlaneEnabled = false
		}
		credential, err := lifecycle.CreateUser(r.Context(), created.Username, dataPlaneEnabled)
		if err != nil {
			code, status := proxySyncFailure(err)
			s.markProxySyncFailure(w, r, created.Username, code, status)
			return
		}
		responseUser := created
		if trigger != nil {
			if err := trigger.Trigger(created.Username); err != nil {
				// Never suppress the reveal-once secret after Telemt already created it.
				if failed, markErr := proxyuser.MarkSyncError(r.Context(), s.db, created.Username, quotaReconcileUnavailableCode); markErr == nil {
					responseUser = failed
				}
			}
		} else {
			synced, err := proxyuser.MarkSynced(r.Context(), s.db, created.Username)
			if err != nil {
				s.writeInternalError(w, r)
				return
			}
			responseUser = synced
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusCreated, struct {
			User   proxyuser.User `json:"user"`
			Secret string         `json:"secret"`
		}{User: responseUser, Secret: credential.Secret})
	}
}

func (s *Server) handleProxyUserEnabled(lifecycle proxyUserLifecycle, trigger quotaReconcileTrigger, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requireAdminAPI(w, r, true) {
			return
		}
		username := r.PathValue("username")
		if err := proxyuser.ValidateUsername(username); err != nil {
			writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "PROXY_USER_INVALID", Message: "The proxy username is invalid.", Instance: r.URL.Path})
			return
		}
		pending, err := proxyuser.SetDesiredEnabled(r.Context(), s.db, username, enabled)
		if errors.Is(err, proxyuser.ErrNotFound) {
			writeProblem(w, Problem{Type: "about:blank", Title: "Not Found", Status: http.StatusNotFound, Code: "PROXY_USER_NOT_FOUND", Message: "The proxy user was not found.", Instance: r.URL.Path})
			return
		}
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if lifecycle == nil {
			s.markProxySyncFailure(w, r, pending.Username, "TELEMT_NOT_CONFIGURED", http.StatusServiceUnavaile)
			return
		}
		dataPlaneEnabled := enabled
		if trigger != nil {
			// Never enable before the current quota projection has been reconciled.
			dataPlaneEnabled = false
		}
		if _, err := lifecycle.SetUserEnabled(r.Context(), pending.Username, dataPlaneEnabled); err != nil {
			code, status := proxySyncFailure(err)
			s.markProxySyncFailure(w, r, pending.Username, code, status)
			return
		}
		if trigger != nil {
			if err := trigger.Trigger(pending.Username); err != nil {
				s.markProxySyncFailure(w, r, pending.Username, quotaReconcileUnavailableCode, http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			writeJSON(w, http.StatusOK, map[string]any{"user": pending})
			return
		}

		synced, err := proxyuser.MarkSynced(r.Context(), s.db, pending.Username)
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, map[string]any{"user": synced})
	}
}

func (s *Server) handleProxyUserRotate(lifecycle proxyUserLifecycle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requireAdminAPI(w, r, true) {
			return
		}
		username := r.PathValue("username")
		if err := proxyuser.ValidateUsername(username); err != nil {
			writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "PROXY_USER_INVALID", Message: "The proxy username is invalid.", Instance: r.URL.Path})
			return
		}
		user, err := proxyuser.Get(r.Context(), s.db, username)
		if errors.Is(err, proxyuser.ErrNotFound) {
			writeProblem(w, Problem{Type: "about:blank", Title: "Not Found", Status: http.StatusNotFound, Code: "PROXY_USER_NOT_FOUND", Message: "The proxy user was not found.", Instance: r.URL.Path})
			return
		}
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if lifecycle == nil {
			s.markProxySyncFailure(w, r, user.Username, "TELEMT_NOT_CONFIGURED", http.StatusServiceUnavailable)
			return
		}
		credential, err := lifecycle.RotateUserSecret(r.Context(), user.Username)
		if err != nil {
			code, status := proxySyncFailure(err)
			s.markProxySyncFailure(w, r, user.Username, code, status)
			return
		}
		synced, err := proxyuser.MarkSynced(r.Context(), s.db, user.Username)
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, struct {
			User   proxyuser.User `json:"user"`
			Secret string         `json:"secret"`
		}{User: synced, Secret: credential.Secret})
	}
}

func (s *Server) requireAdminAPI(w http.ResponseWriter, r *http.Request, mutation bool) bool {
	_, token, ok, err := s.currentSession(r)
	if err != nil {
		s.writeInternalError(w, r)
		return false
	}
	if !ok {
		writeProblem(w, Problem{Type: "about:blank", Title: "Unauthorized", Status: http.StatusUnauthorized, Code: "AUTH_REQUIRED", Message: "Administrator authentication is required.", Instance: r.URL.Path})
		return false
	}
	if mutation && !secureEqual(sessionCSRF(token), r.Header.Get("X-CSRF-Token")) {
		writeProblem(w, Problem{Type: "about:blank", Title: "Forbidden", Status: http.StatusForbidden, Code: "CSRF_INVALID", Message: "The request could not be verified.", Instance: r.URL.Path})
		return false
	}
	return true
}

func decodeProxyJSON(w http.ResponseWriter, r *http.Request, output any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxProxyUserBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "The JSON request is invalid.", Instance: r.URL.Path})
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "The JSON request is invalid.", Instance: r.URL.Path})
		return false
	}
	return true
}

func (s *Server) markProxySyncFailure(w http.ResponseWriter, r *http.Request, username, code string, status int) {
	if _, err := proxyuser.MarkSyncError(r.Context(), s.db, username, code); err != nil {
		s.writeInternalError(w, r)
		return
	}
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: "The desired proxy-user state was saved, but the proxy data plane has not converged yet.", Instance: r.URL.Path})
}

func proxySyncFailure(err error) (string, int) {
	switch telemt.FailureCodeOf(err) {
	case telemt.FailureUnauthorized:
		return "TELEMT_UNAUTHORIZED", http.StatusBadGateway
	case telemt.FailureForbidden:
		return "TELEMT_FORBIDDEN", http.StatusBadGateway
	case telemt.FailureNotFound:
		return "TELEMT_NOT_FOUND", http.StatusBadGateway
	case telemt.FailureConflict:
		return "TELEMT_CONFLICT", http.StatusConflict
	case telemt.FailureRejected:
		return "TELEMT_REJECTED", http.StatusBadGateway
	case telemt.FailureInvalidOutput:
		return "TELEMT_INVALID_RESPONSE", http.StatusBadGateway
	case telemt.FailureUnavailable:
		return "TELEMT_UNAVAILABLE", http.StatusServiceUnavailable
	default:
		return "TELEMT_UNAVAILABLE", http.StatusServiceUnavailable
	}
}
