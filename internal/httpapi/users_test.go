package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
	"github.com/smorad3363/teleproxy/internal/useradmin"
)

func TestUserInventoryAPIRequiresAuthenticationAndIsNoStore(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/users", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized users = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	authorized := performJSON(server.Handler(), http.MethodGet, "/api/users", "", cookie, "")
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized users = %d %s", authorized.Code, authorized.Body.String())
	}
	if got := authorized.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("users Cache-Control = %q", got)
	}
	if !strings.Contains(authorized.Body.String(), `"users":[]`) {
		t.Fatalf("empty users body = %s", authorized.Body.String())
	}
}

func TestUserInventoryAPIPaginatesWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	firstUser, err := telegramuser.Resolve(ctx, db, 9201, time.Now().UTC().Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	secondUser, err := telegramuser.Resolve(ctx, db, 9202, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := proxyprovision.Prepare(ctx, db, secondUser.User.ProxyUsername, [32]byte{1}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("proxyprovision.Prepare() created = false")
	}

	beforeTelegram := countHTTPRows(t, db, "telegram_users")
	beforeProxy := countHTTPRows(t, db, "proxy_users")
	beforeProvisioning := countHTTPRows(t, db, "proxy_user_provisioning")
	beforeReferrals := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	firstResponse := performJSON(server.Handler(), http.MethodGet, "/api/users?limit=1", "", cookie, "")
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first users page = %d %s", firstResponse.Code, firstResponse.Body.String())
	}
	if strings.Contains(firstResponse.Body.String(), "secret_sha256") {
		t.Fatalf("first users page exposed provisioning secret digest: %s", firstResponse.Body.String())
	}
	var firstPage struct {
		Users        []useradmin.Entry `json:"users"`
		NextBeforeID *int64            `json:"next_before_id"`
	}
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Users) != 1 || firstPage.Users[0].TelegramUserID != secondUser.User.ID || firstPage.Users[0].TelegramID != 9202 {
		t.Fatalf("first users page = %#v", firstPage)
	}
	if firstPage.Users[0].ProvisioningPhase == nil || *firstPage.Users[0].ProvisioningPhase != proxyprovision.PhasePrepared {
		t.Fatalf("first provisioning phase = %v, want prepared", firstPage.Users[0].ProvisioningPhase)
	}
	if firstPage.NextBeforeID == nil || *firstPage.NextBeforeID != secondUser.User.ID {
		t.Fatalf("first next_before_id = %v", firstPage.NextBeforeID)
	}

	secondResponse := performJSON(server.Handler(), http.MethodGet, "/api/users?limit=1&before_id="+strconv.FormatInt(*firstPage.NextBeforeID, 10), "", cookie, "")
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second users page = %d %s", secondResponse.Code, secondResponse.Body.String())
	}
	var secondPage struct {
		Users        []useradmin.Entry `json:"users"`
		NextBeforeID *int64            `json:"next_before_id"`
	}
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Users) != 1 || secondPage.Users[0].TelegramUserID != firstUser.User.ID || secondPage.NextBeforeID != nil {
		t.Fatalf("second users page = %#v", secondPage)
	}
	if secondPage.Users[0].ProvisioningPhase != nil {
		t.Fatalf("second provisioning phase = %v, want nil", secondPage.Users[0].ProvisioningPhase)
	}

	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "proxy_user_provisioning") != beforeProvisioning || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("user inventory API mutated authoritative state")
	}
}

func TestUserInventoryAPIFiltersExactProvisioningPhaseWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	now := time.Now().UTC().Truncate(time.Second)

	unprovisioned, err := telegramuser.Resolve(ctx, db, 9251, now.Add(-3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := telegramuser.Resolve(ctx, db, 9252, now.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	collision, err := telegramuser.Resolve(ctx, db, 9253, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	preparedDigest := [32]byte{2}
	if _, created, err := proxyprovision.Prepare(ctx, db, prepared.User.ProxyUsername, preparedDigest, now); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("prepared proxyprovision.Prepare() created = false")
	}
	collisionDigest := [32]byte{3}
	if _, created, err := proxyprovision.Prepare(ctx, db, collision.User.ProxyUsername, collisionDigest, now); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("collision proxyprovision.Prepare() created = false")
	}
	if _, err := proxyprovision.MarkCollision(ctx, db, collision.User.ProxyUsername, collisionDigest, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	beforeTelegram := countHTTPRows(t, db, "telegram_users")
	beforeProxy := countHTTPRows(t, db, "proxy_users")
	beforeProvisioning := countHTTPRows(t, db, "proxy_user_provisioning")
	beforeReferrals := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	for _, tc := range []struct {
		path       string
		wantUserID int64
		wantPhase  *proxyprovision.Phase
	}{
		{path: "/api/users?provisioning_phase=prepared", wantUserID: prepared.User.ID, wantPhase: phasePointer(proxyprovision.PhasePrepared)},
		{path: "/api/users?provisioning_phase=collision", wantUserID: collision.User.ID, wantPhase: phasePointer(proxyprovision.PhaseCollision)},
		{path: "/api/users?provisioning_phase=owned"},
		{path: "/api/users?telegram_id=" + strconv.FormatInt(unprovisioned.User.TelegramID, 10) + "&provisioning_phase=prepared"},
	} {
		response := performJSON(server.Handler(), http.MethodGet, tc.path, "", cookie, "")
		if response.Code != http.StatusOK {
			t.Fatalf("filtered users %q = %d %s", tc.path, response.Code, response.Body.String())
		}
		var page struct {
			Users []useradmin.Entry `json:"users"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if tc.wantUserID == 0 {
			if len(page.Users) != 0 {
				t.Fatalf("filtered users %q = %#v, want empty", tc.path, page.Users)
			}
			continue
		}
		if len(page.Users) != 1 || page.Users[0].TelegramUserID != tc.wantUserID || page.Users[0].ProvisioningPhase == nil || tc.wantPhase == nil || *page.Users[0].ProvisioningPhase != *tc.wantPhase {
			t.Fatalf("filtered users %q = %#v", tc.path, page.Users)
		}
	}

	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "proxy_user_provisioning") != beforeProvisioning || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("provisioning phase filter mutated authoritative state")
	}
}

func TestUserInventoryAPIRejectsInvalidPagination(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	for _, path := range []string{
		"/api/users?before_id=0",
		"/api/users?before_id=-1",
		"/api/users?before_id=nope",
		"/api/users?before_id=1&before_id=2",
		"/api/users?limit=0",
		"/api/users?limit=" + strconv.Itoa(useradmin.MaxListLimit+1),
		"/api/users?provisioning_phase=",
		"/api/users?provisioning_phase=PREPARED",
		"/api/users?provisioning_phase=unknown",
		"/api/users?provisioning_phase=prepared&provisioning_phase=owned",
		"/api/users?unknown=1",
	} {
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"USER_INVENTORY_INVALID"`) {
			t.Fatalf("invalid users query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func phasePointer(phase proxyprovision.Phase) *proxyprovision.Phase {
	return &phase
}

func authenticatedUserInventoryAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie) {
	t.Helper()
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}
}
