package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

type fakeQuotaTrigger struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (f *fakeQuotaTrigger) Trigger(username string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, username)
	return f.err
}

func (f *fakeQuotaTrigger) callCount(username string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, call := range f.calls {
		if call == username {
			count++
		}
	}
	return count
}

func TestManualQuotaReconcileRequiresAuthCSRFAndQueues(t *testing.T) {
	db := testDB(t)
	if _, err := proxyuser.Create(context.Background(), db, "alice", true); err != nil {
		t.Fatal(err)
	}
	trigger := &fakeQuotaTrigger{}
	server, cookie, csrf := authenticatedProxyAPIWithTrigger(t, db, fakeProxyLifecycle{}, trigger)

	unauthorized := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/reconcile", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}
	missingCSRF := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/reconcile", "", cookie, "")
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d", missingCSRF.Code)
	}
	queued := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/reconcile", "", cookie, csrf)
	if queued.Code != http.StatusAccepted || !strings.Contains(queued.Body.String(), `"status":"queued"`) {
		t.Fatalf("queued status/body = %d %s", queued.Code, queued.Body.String())
	}
	if trigger.callCount("alice") != 1 {
		t.Fatalf("trigger calls = %d, want 1", trigger.callCount("alice"))
	}
	if strings.Contains(strings.ToLower(queued.Body.String()), "secret") {
		t.Fatalf("manual reconcile leaked secret field: %s", queued.Body.String())
	}
}

func TestManualQuotaReconcileUnavailableUsesNarrowErrorCode(t *testing.T) {
	db := testDB(t)
	if _, err := proxyuser.Create(context.Background(), db, "alice", true); err != nil {
		t.Fatal(err)
	}
	trigger := &fakeQuotaTrigger{err: errors.New("raw upstream secret-token")}
	server, cookie, csrf := authenticatedProxyAPIWithTrigger(t, db, fakeProxyLifecycle{}, trigger)

	response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/reconcile", "", cookie, csrf)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"QUOTA_RECONCILE_UNAVAILABLE"`) {
		t.Fatalf("status/body = %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret-token") {
		t.Fatalf("raw trigger error leaked: %s", response.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.SyncState != proxyuser.SyncError || stored.LastErrorCode != quotaReconcileUnavailableCode {
		t.Fatalf("stored = %#v", stored)
	}
}

func TestCreateWithRunnerStaysPendingUntilQuotaReconciles(t *testing.T) {
	db := testDB(t)
	lifecycle := fakeProxyLifecycle{create: func(_ context.Context, username string, enabled bool) (telemt.Credential, error) {
		if enabled {
			t.Fatal("Telemt user was created enabled before quota bootstrap")
		}
		return telemt.Credential{User: telemt.User{Username: username, Enabled: false, InRuntime: true}, Secret: proxyUserTestSecret}, nil
	}}
	trigger := &fakeQuotaTrigger{}
	server, cookie, csrf := authenticatedProxyAPIWithTrigger(t, db, lifecycle, trigger)

	response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users", `{"username":"alice"}`, cookie, csrf)
	if response.Code != http.StatusCreated || strings.Count(response.Body.String(), proxyUserTestSecret) != 1 {
		t.Fatalf("create status/body = %d %s", response.Code, response.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !stored.DesiredEnable || stored.SyncState != proxyuser.SyncPending {
		t.Fatalf("stored before async reconciliation = %#v", stored)
	}
	if trigger.callCount("alice") != 1 {
		t.Fatalf("trigger calls = %d, want 1", trigger.callCount("alice"))
	}
}

func TestCreatePreservesRevealOnceSecretWhenQueueFails(t *testing.T) {
	db := testDB(t)
	lifecycle := fakeProxyLifecycle{create: func(_ context.Context, username string, enabled bool) (telemt.Credential, error) {
		if enabled {
			t.Fatal("Telemt user was created enabled before quota bootstrap")
		}
		return telemt.Credential{User: telemt.User{Username: username, Enabled: enabled, InRuntime: true}, Secret: proxyUserTestSecret}, nil
	}}
	trigger := &fakeQuotaTrigger{err: errors.New("runner closed")}
	server, cookie, csrf := authenticatedProxyAPIWithTrigger(t, db, lifecycle, trigger)

	response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users", `{"username":"alice"}`, cookie, csrf)
	if response.Code != http.StatusCreated || strings.Count(response.Body.String(), proxyUserTestSecret) != 1 {
		t.Fatalf("create status/body = %d %s", response.Code, response.Body.String())
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.SyncState != proxyuser.SyncError || stored.LastErrorCode != quotaReconcileUnavailableCode {
		t.Fatalf("stored after queue failure = %#v", stored)
	}
	if trigger.callCount("alice") != 1 {
		t.Fatalf("trigger calls = %d, want 1", trigger.callCount("alice"))
	}
}

func TestEnableDisableQueueQuotaReconciliationAfterLifecycleSuccess(t *testing.T) {
	db := testDB(t)
	if _, err := proxyuser.Create(context.Background(), db, "alice", false); err != nil {
		t.Fatal(err)
	}
	var lifecycleEnabled []bool
	lifecycle := fakeProxyLifecycle{enable: func(_ context.Context, username string, enabled bool) (telemt.User, error) {
		lifecycleEnabled = append(lifecycleEnabled, enabled)
		if enabled {
			t.Fatal("runner-backed desired enable reached Telemt before quota reconciliation")
		}
		return telemt.User{Username: username, Enabled: enabled, InRuntime: true}, nil
	}}
	trigger := &fakeQuotaTrigger{}
	server, cookie, csrf := authenticatedProxyAPIWithTrigger(t, db, lifecycle, trigger)

	for _, action := range []string{"enable", "disable"} {
		response := performJSON(server.Handler(), http.MethodPost, "/api/proxy/users/alice/"+action, "", cookie, csrf)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status/body = %d %s", action, response.Code, response.Body.String())
		}
	}
	if trigger.callCount("alice") != 2 {
		t.Fatalf("trigger calls = %d, want 2", trigger.callCount("alice"))
	}
	if len(lifecycleEnabled) != 2 || lifecycleEnabled[0] || lifecycleEnabled[1] {
		t.Fatalf("Telemt enabled args = %v, want [false false]", lifecycleEnabled)
	}
	stored, err := proxyuser.Get(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnable || stored.SyncState != proxyuser.SyncPending {
		t.Fatalf("stored after queued disable = %#v", stored)
	}
}

func authenticatedProxyAPIWithTrigger(t *testing.T, db *sql.DB, lifecycle proxyUserLifecycle, trigger quotaReconcileTrigger) (*Server, []*http.Cookie, string) {
	t.Helper()
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := newWithProxyDependenciesAndReconciler(db, Options{}, nil, lifecycle, trigger)
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}, sessionCSRF(token)
}
