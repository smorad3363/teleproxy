package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/auth"
)

const (
	sessionCookieName   = "tp_session"
	loginCSRFCookieName = "tp_login_csrf"
	maxLoginBodyBytes   = 16 << 10
)

type Options struct {
	CookieSecure    bool
	SessionTTL      time.Duration
	LoginMaxFailure int
	LoginWindow     time.Duration
}

type Server struct {
	mux          *http.ServeMux
	db           *sql.DB
	cookieSecure bool
	sessionTTL   time.Duration
	loginLimiter *loginLimiter
}

func New(db *sql.DB, options Options) *Server {
	if options.SessionTTL <= 0 {
		options.SessionTTL = 12 * time.Hour
	}
	s := &Server{
		mux:          http.NewServeMux(),
		db:           db,
		cookieSecure: options.CookieSecure,
		sessionTTL:   options.SessionTTL,
		loginLimiter: newLoginLimiter(options.LoginMaxFailure, options.LoginWindow),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("GET /login", s.handleLoginGet)
	s.mux.HandleFunc("POST /login", s.handleLoginPost)
	s.mux.HandleFunc("POST /logout", s.handleLogout)
	s.mux.HandleFunc("GET /{$}", s.handleDashboard)
	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		writeProblem(w, Problem{Type: "about:blank", Title: "Service Unavailable", Status: http.StatusServiceUnavailable, Code: "DATABASE_NOT_READY", Message: "The Control Plane is not ready."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	if _, _, ok, err := s.currentSession(r); err != nil {
		s.writeInternalError(w, r)
		return
	} else if ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	csrf, _, err := auth.NewSessionToken()
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	s.setLoginCSRFCookie(w, csrf)
	renderLogin(w, http.StatusOK, csrf, "")
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeInternalError(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBodyBytes)
	if err := r.ParseForm(); err != nil {
		writeProblem(w, Problem{Type: "about:blank", Title: "Bad Request", Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "The login request is invalid.", Instance: r.URL.Path})
		return
	}

	csrfCookie, err := r.Cookie(loginCSRFCookieName)
	if err != nil || !secureEqual(csrfCookie.Value, r.FormValue("csrf")) {
		writeProblem(w, Problem{Type: "about:blank", Title: "Forbidden", Status: http.StatusForbidden, Code: "CSRF_INVALID", Message: "The request could not be verified.", Instance: r.URL.Path})
		return
	}

	peer := directPeerIP(r.RemoteAddr)
	now := time.Now().UTC()
	if !s.loginLimiter.Allow(peer, now) {
		w.Header().Set("Retry-After", "60")
		writeProblem(w, Problem{Type: "about:blank", Title: "Too Many Requests", Status: http.StatusTooManyRequests, Code: "LOGIN_RATE_LIMITED", Message: "Too many login attempts. Try again later.", Instance: r.URL.Path})
		return
	}

	administrator, err := admin.Authenticate(r.Context(), s.db, r.FormValue("username"), r.FormValue("password"))
	if errors.Is(err, admin.ErrInvalidCredentials) {
		s.loginLimiter.Failure(peer, now)
		renderLogin(w, http.StatusUnauthorized, csrfCookie.Value, "Invalid username or password.")
		return
	}
	if err != nil {
		s.writeInternalError(w, r)
		return
	}

	token, expiresAt, err := admin.CreateSession(r.Context(), s.db, administrator.ID, s.sessionTTL)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	s.loginLimiter.Reset(peer)
	s.setSessionCookie(w, token, expiresAt)
	s.clearLoginCSRFCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	session, token, ok, err := s.currentSession(r)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	if !ok {
		s.clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = dashboardTemplate.Execute(w, struct {
		Username string
		CSRF     string
	}{Username: session.Admin.Username, CSRF: sessionCSRF(token)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, token, ok, err := s.currentSession(r)
	_ = session
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	if !ok {
		s.clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil || !secureEqual(sessionCSRF(token), r.FormValue("csrf")) {
		writeProblem(w, Problem{Type: "about:blank", Title: "Forbidden", Status: http.StatusForbidden, Code: "CSRF_INVALID", Message: "The request could not be verified.", Instance: r.URL.Path})
		return
	}
	if err := admin.RevokeSession(r.Context(), s.db, token); err != nil {
		s.writeInternalError(w, r)
		return
	}
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) currentSession(r *http.Request) (admin.Session, string, bool, error) {
	if s.db == nil {
		return admin.Session{}, "", false, nil
	}
	cookie, err := r.Cookie(sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return admin.Session{}, "", false, nil
	}
	if err != nil {
		return admin.Session{}, "", false, err
	}
	session, err := admin.SessionByToken(r.Context(), s.db, cookie.Value)
	if errors.Is(err, admin.ErrSessionNotFound) {
		return admin.Session{}, "", false, nil
	}
	if err != nil {
		return admin.Session{}, "", false, err
	}
	return session, cookie.Value, true, nil
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expiresAt, HttpOnly: true, Secure: s.cookieSecure, SameSite: http.SameSiteStrictMode})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cookieSecure, SameSite: http.SameSiteStrictMode})
}

func (s *Server) setLoginCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: loginCSRFCookieName, Value: token, Path: "/login", MaxAge: 600, HttpOnly: true, Secure: s.cookieSecure, SameSite: http.SameSiteStrictMode})
}

func (s *Server) clearLoginCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: loginCSRFCookieName, Value: "", Path: "/login", MaxAge: -1, HttpOnly: true, Secure: s.cookieSecure, SameSite: http.SameSiteStrictMode})
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, Problem{Type: "about:blank", Title: "Not Found", Status: http.StatusNotFound, Code: "RESOURCE_NOT_FOUND", Message: "The requested resource was not found.", Instance: r.URL.Path})
}

func (s *Server) writeInternalError(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, Problem{Type: "about:blank", Title: "Internal Server Error", Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "The request could not be completed.", Instance: r.URL.Path})
}

func renderLogin(w http.ResponseWriter, status int, csrf, message string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = loginTemplate.Execute(w, struct {
		CSRF  string
		Error string
	}{CSRF: csrf, Error: message})
}

func sessionCSRF(sessionToken string) string {
	mac := hmac.New(sha256.New, []byte(sessionToken))
	_, _ = mac.Write([]byte("teleproxy:csrf:v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func secureEqual(a, b string) bool {
	return a != "" && b != "" && hmac.Equal([]byte(a), []byte(b))
}

func directPeerIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}
	if remoteAddr == "" {
		return "unknown"
	}
	return remoteAddr
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
