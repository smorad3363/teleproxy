package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

const proxyUserTestSecret = "00112233445566778899aabbccddeeff"

type fakeProxyLifecycle struct {
	create func(context.Context, string, bool) (telemt.Credential, error)
	enable func(context.Context, string, bool) (telemt.User, error)
	rotate func(context.Context, string) (telemt.Credential, error)
}

func (f fakeProxyLifecycle) CreateUser(ctx context.Context, username string, enabled bool) (telemt.Credential, error) {
	if f.create == nil {
		return telemt.Credential{}, &telemt.APIError{Code: telemt.FailureUnavailable}
	}
	return f.create(ctx, username, enabled)
}

func (f fakeProxyLifecycle) SetUserEnabled(ctx context.Context, username string, enabled bool) (telemt.User, error) {
	if f.enable == nil {
		return telemt.User{}, &telemt.APIError{Code: telemt.FailureUnavailable}
	}
	return f.enable(ctx, username, enabled)
}

func (f fakeProxyLifecycle) RotateUserSecret(ctx context.Context, username string) (telemt.Credential, error) {
	if f.rotate == nil {
		return telemt.Credential{}, &telemt.APIError{Code: telemt.FailureUnavailable}
	}
	return f.rotate(ctx, username)
}

func TestProxyUsersRequireAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedProxyAPI(t, db, fakeProxyLifecycle{})

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/proxy/users", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}

	missingCSRF := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users", `{"username":"alice"}`, cookie, "")
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d, body=%s", missingCSRF.Code, missingCSRF.Body.String())
	}
	users, err := proxyuser.List(context.Background(), db)
	if err != nil || len(users) != 0 {
		t.Fatalf("users after rejected mutation = %#v, %v", users, err)
	}
}

func TestProxyUserCreateAndListRevealSecretOnlyOnCreate(t *testing.T) {
	db := testDB(t)
	lifecycle := fakeProxyLifecycle{create: func(_ context.Context, username string, enabled bool) (telemt.Credential, error) {
		if username != "alice" || !enabled {
			t.Fatalf("create args = %q, %v", username, enabled)
		}
		return telemt.Credential{User: telemt.User{Username: username, Enabled: enabled, InRuntime: true}, Secret: proxyUserTestSecret}, nil
	}}
	server, cookie, csrf := authenticatedProxyAPI(t, db, lifecycle)

	created := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users", `{"username":"alice"}`, cookie, csrf)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", created.Code, created.Body.String())
	}
	if strings.Count(created.Body.String(), proxyUserTestSecret) != 1 {
		t.Fatalf("create secret count unexpected: %s", created.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.SyncState != proxyuser.SyncSynced || !stored.DesiredEnable {
		t.Fatalf("stored = %#v", stored)
	}

	listed := performJSON(server.Handler(), http.MethodGet, "/api/proxy/users", "", cookie, "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", listed.Code, listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), proxyUserTestSecret) || strings.Contains(listed.Body.String(), `"secret"`) {
		t.Fatalf("list leaked secret material: %s", listed.Body.String())
	}
	if !strings.Contains(listed.Body.String(), `"username":"alice"`) {
		t.Fatalf("list missing user: %s", listed.Body.String())
	}
}

func TestProxyUserCreateUnavailablePreservesDesiredState(t *testing.T) {
	db := testDB(t)
	lifecycle := fakeProxyLifecycle{create: func(context.Context, string, bool) (telemt.Credential, error) {
		return telemt.Credential{}, &telemt.APIError{Code: telemt.FailureUnavailable}
	}}
	server, cookie, csrf := authenticatedProxyAPI(t, db, lifecycle)

	response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users", `{"username":"alice","enabled":false}`, cookie, csrf)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"TELEMT_UNAVAILABLE"`) || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("unsafe failure body: %s", response.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnable || stored.SyncState != proxyuser.SyncError || stored.LastErrorCode != "TELEMT_UNAVAILABLE" {
		t.Fatalf("stored after failure = %#v", stored)
	}
}

func TestProxyUserDisableUnavailableKeepsDesiredDisable(t *testing.T) {
	db := testDB(t)
	if _, err := proxyuser.Create(context.Background(), db, "alice", true); err != nil {
		t.Fatal(err)
	}
	if _, err := proxyuser.MarkSynced(context.Background(), db, "alice"); err != nil {
		t.Fatal(err)
	}
	lifecycle := fakeProxyLifecycle{enable: func(context.Context, string, bool) (telemt.User, error) {
		return telemt.User{}, &telemt.APIError{Code: telemt.FailureUnavailable}
	}}
	server, cookie, csrf := authenticatedProxyAPI(t, db, lifecycle)

	response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/disable", "", cookie, csrf)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("disable status = %d, body=%s", response.Code, response.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnable || stored.SyncState != proxyuser.SyncError || stored.LastErrorCode != "TELEMT_UNAVAILABLE" {
		t.Fatalf("stored after disable failure = %#v", stored)
	}
}

func TestProxyUserRotateReturnsSecretWithoutChangingDesiredState(t *testing.T) {
	db := testDB(t)
	if _, err := proxyuser.Create(context.Background(), db, "alice", false); err != nil {
		t.Fatal(err)
	}
	lifecycle := fakeProxyLifecycle{rotate: func(_ context.Context, username string) (telemt.Credential, error) {
		return telemt.Credential{User: telemt.User{Username: username, Enabled: false}, Secret: proxyUserTestSecret}, nil
	}}
	server, cookie, csrf := authenticatedProxyAPI(t, db, lifecycle)

	response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/rotate-secret", "", cookie, csrf)
	if response.Code != http.StatusOK || strings.Count(response.Body.String(), proxyUserTestSecret) != 1 {
		t.Fatalf("rotate status/body = %d %s", response.Code, response.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnable || stored.SyncState != proxyuser.SyncSynced {
		t.Fatalf("stored after rotate = %#v", stored)
	}
}

func authenticatedProxyAPI(t *testing.T, db *sql.DB, lifecycle proxyUserLifecycle) (*Server, []*http.Cookie, string) {
	t.Helper()
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := newWithProxyDependencies(db, Options{}, nil, lifecycle)
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}, sessionCSRF(token)
}

func performJSON(handler http.Handler, method, path, body string, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.10:4242"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
