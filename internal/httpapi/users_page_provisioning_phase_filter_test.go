package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestUserPageFiltersExactProvisioningPhaseAndPreservesPaginationWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	now := time.Now().UTC().Truncate(time.Second)

	unprovisioned, err := telegramuser.Resolve(ctx, db, 9351, now.Add(-4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	preparedOlder, err := telegramuser.Resolve(ctx, db, 9352, now.Add(-3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	preparedNewer, err := telegramuser.Resolve(ctx, db, 9353, now.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	collision, err := telegramuser.Resolve(ctx, db, 9354, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	for index, resolved := range []telegramuser.ResolveResult{preparedOlder, preparedNewer} {
		digest := [32]byte{byte(index + 1)}
		if _, created, err := proxyprovision.Prepare(ctx, db, resolved.User.ProxyUsername, digest, now); err != nil {
			t.Fatal(err)
		} else if !created {
			t.Fatalf("prepared user %d proxyprovision.Prepare() created = false", resolved.User.TelegramID)
		}
	}
	collisionDigest := [32]byte{9}
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

	first := perform(server.Handler(), http.MethodGet, "/users?limit=1&provisioning_phase=prepared", nil, cookie)
	if first.Code != http.StatusOK {
		t.Fatalf("prepared users page = %d %s", first.Code, first.Body.String())
	}
	firstBody := first.Body.String()
	for _, want := range []string{
		">9353<",
		`<option value="prepared" selected>prepared</option>`,
		`href="/users?before_id=` + strconv.FormatInt(preparedNewer.User.ID, 10) + `&amp;limit=1&amp;provisioning_phase=prepared"`,
	} {
		if !strings.Contains(firstBody, want) {
			t.Fatalf("prepared users page missing %q: %s", want, firstBody)
		}
	}
	for _, unwanted := range []string{
		">" + strconv.FormatInt(unprovisioned.User.TelegramID, 10) + "<",
		">" + strconv.FormatInt(preparedOlder.User.TelegramID, 10) + "<",
		">" + strconv.FormatInt(collision.User.TelegramID, 10) + "<",
		"secret_sha256",
	} {
		if strings.Contains(firstBody, unwanted) {
			t.Fatalf("prepared users page unexpectedly contains %q: %s", unwanted, firstBody)
		}
	}

	older := perform(server.Handler(), http.MethodGet, "/users?before_id="+strconv.FormatInt(preparedNewer.User.ID, 10)+"&limit=1&provisioning_phase=prepared", nil, cookie)
	if older.Code != http.StatusOK || !strings.Contains(older.Body.String(), ">9352<") || strings.Contains(older.Body.String(), ">9353<") || strings.Contains(older.Body.String(), ">9354<") {
		t.Fatalf("prepared users older page mismatch: %d %s", older.Code, older.Body.String())
	}

	collisionPage := perform(server.Handler(), http.MethodGet, "/users?provisioning_phase=collision", nil, cookie)
	if collisionPage.Code != http.StatusOK || !strings.Contains(collisionPage.Body.String(), ">9354<") || !strings.Contains(collisionPage.Body.String(), `<option value="collision" selected>collision</option>`) || strings.Contains(collisionPage.Body.String(), ">9353<") {
		t.Fatalf("collision users page mismatch: %d %s", collisionPage.Code, collisionPage.Body.String())
	}

	ownedPage := perform(server.Handler(), http.MethodGet, "/users?provisioning_phase=owned", nil, cookie)
	if ownedPage.Code != http.StatusOK || !strings.Contains(ownedPage.Body.String(), "No users match the current filters.") || !strings.Contains(ownedPage.Body.String(), `<option value="owned" selected>owned</option>`) {
		t.Fatalf("owned users empty page mismatch: %d %s", ownedPage.Code, ownedPage.Body.String())
	}

	for _, path := range []string{
		"/users?provisioning_phase=",
		"/users?provisioning_phase=PREPARED",
		"/users?provisioning_phase=unknown",
		"/users?provisioning_phase=prepared&provisioning_phase=owned",
	} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"USER_INVENTORY_INVALID"`) {
			t.Fatalf("invalid provisioning phase page query %q = %d %s", path, response.Code, response.Body.String())
		}
	}

	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "proxy_user_provisioning") != beforeProvisioning || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("provisioning phase page filter mutated authoritative state")
	}
}
