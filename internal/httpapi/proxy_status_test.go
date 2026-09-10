package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

type fakeProxyHealth func(context.Context) telemt.Health

func (f fakeProxyHealth) Health(ctx context.Context) telemt.Health { return f(ctx) }

func TestProxyStatusRequiresAdminSession(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyHealth(db, Options{}, fakeProxyHealth(func(context.Context) telemt.Health {
		return telemt.Health{State: telemt.StateHealthy}
	}))

	response := perform(server.Handler(), http.MethodGet, "/api/system/proxy", nil, nil)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestProxyStatusReturnsSafeHealthState(t *testing.T) {
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyHealth(db, Options{}, fakeProxyHealth(func(context.Context) telemt.Health {
		return telemt.Health{State: telemt.StateUnavailable}
	}))

	response := perform(server.Handler(), http.MethodGet, "/api/system/proxy", nil, []*http.Cookie{{Name: sessionCookieName, Value: token}})
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"state":"unavailable"`) {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestReadyDoesNotDependOnProxyHealth(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyHealth(db, Options{}, fakeProxyHealth(func(context.Context) telemt.Health {
		return telemt.Health{State: telemt.StateUnavailable}
	}))
	response := perform(server.Handler(), http.MethodGet, "/readyz", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("ready status = %d, body=%s", response.Code, response.Body.String())
	}
}
