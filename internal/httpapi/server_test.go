package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/database"
)

var csrfPattern = regexp.MustCompile(`name="csrf" value="([^"]+)"`)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	New(nil, Options{}).Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyWithDatabase(t *testing.T) {
	db := testDB(t)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	New(db, Options{}).Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotFoundUsesProblemDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	New(nil, Options{}).Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestLoginDashboardAndLogout(t *testing.T) {
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	server := New(db, Options{SessionTTL: time.Hour, CookieSecure: true})

	unauth := perform(server.Handler(), http.MethodGet, "/", nil, nil)
	if unauth.Code != http.StatusSeeOther || unauth.Header().Get("Location") != "/login" {
		t.Fatalf("unauth dashboard = %d location=%q", unauth.Code, unauth.Header().Get("Location"))
	}

	loginPage := perform(server.Handler(), http.MethodGet, "/login", nil, nil)
	csrf := extractCSRF(t, loginPage.Body.String())
	loginCSRFCookie := namedCookie(t, loginPage.Result().Cookies(), loginCSRFCookieName)

	wrongForm := url.Values{"username": {"admin"}, "password": {"wrong-password"}, "csrf": {csrf}}
	wrong := perform(server.Handler(), http.MethodPost, "/login", wrongForm, []*http.Cookie{loginCSRFCookie})
	if wrong.Code != http.StatusUnauthorized || !strings.Contains(wrong.Body.String(), "Invalid username or password.") {
		t.Fatalf("wrong login status/body = %d %q", wrong.Code, wrong.Body.String())
	}

	goodForm := url.Values{"username": {"admin"}, "password": {"generated-admin-password-123"}, "csrf": {csrf}}
	good := perform(server.Handler(), http.MethodPost, "/login", goodForm, []*http.Cookie{loginCSRFCookie})
	if good.Code != http.StatusSeeOther || good.Header().Get("Location") != "/" {
		t.Fatalf("good login = %d location=%q", good.Code, good.Header().Get("Location"))
	}
	sessionCookie := namedCookie(t, good.Result().Cookies(), sessionCookieName)
	if !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie flags = %#v", sessionCookie)
	}

	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, []*http.Cookie{sessionCookie})
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), owner.Username) {
		t.Fatalf("dashboard = %d %q", dashboard.Code, dashboard.Body.String())
	}
	logoutCSRF := extractCSRF(t, dashboard.Body.String())

	badLogout := perform(server.Handler(), http.MethodPost, "/logout", url.Values{"csrf": {"wrong"}}, []*http.Cookie{sessionCookie})
	if badLogout.Code != http.StatusForbidden {
		t.Fatalf("bad logout status = %d", badLogout.Code)
	}

	logout := perform(server.Handler(), http.MethodPost, "/logout", url.Values{"csrf": {logoutCSRF}}, []*http.Cookie{sessionCookie})
	if logout.Code != http.StatusSeeOther || logout.Header().Get("Location") != "/login" {
		t.Fatalf("logout = %d location=%q", logout.Code, logout.Header().Get("Location"))
	}
	if _, err := admin.SessionByToken(context.Background(), db, sessionCookie.Value); !errors.Is(err, admin.ErrSessionNotFound) {
		t.Fatalf("session still active after logout: %v", err)
	}
}

func TestLoginRateLimit(t *testing.T) {
	db := testDB(t)
	if _, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123"); err != nil {
		t.Fatal(err)
	}
	server := New(db, Options{LoginMaxFailure: 2, LoginWindow: time.Hour})
	page := perform(server.Handler(), http.MethodGet, "/login", nil, nil)
	csrf := extractCSRF(t, page.Body.String())
	csrfCookie := namedCookie(t, page.Result().Cookies(), loginCSRFCookieName)
	form := url.Values{"username": {"admin"}, "password": {"wrong-password"}, "csrf": {csrf}}

	for i := 0; i < 2; i++ {
		response := perform(server.Handler(), http.MethodPost, "/login", form, []*http.Cookie{csrfCookie})
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d status = %d", i+1, response.Code)
		}
	}
	limited := perform(server.Handler(), http.MethodPost, "/login", form, []*http.Cookie{csrfCookie})
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d, want 429", limited.Code)
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func perform(handler http.Handler, method, path string, form url.Values, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	req.RemoteAddr = "192.0.2.10:4242"
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func extractCSRF(t *testing.T, body string) string {
	t.Helper()
	match := csrfPattern.FindStringSubmatch(body)
	if len(match) != 2 || match[1] == "" {
		t.Fatalf("CSRF token not found in body: %q", body)
	}
	return match[1]
}

func namedCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.MaxAge >= 0 {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found: %#v", name, cookies)
	return nil
}
