package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/referral"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestUserPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/users", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated users page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "Users") {
		t.Fatalf("unauthenticated users page rendered inventory markup: %s", response.Body.String())
	}
}

func TestUserPageRendersAuthoritativeInventoryAndPaginationWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	now := time.Now().UTC().Truncate(time.Second)

	invitee, err := telegramuser.Resolve(ctx, db, 9301, now.Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	inviter, err := telegramuser.Resolve(ctx, db, 9302, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := proxyuser.SetDesiredEnabled(ctx, db, inviter.User.ProxyUsername, false); err != nil {
		t.Fatal(err)
	}
	if _, err := proxyuser.MarkSyncError(ctx, db, inviter.User.ProxyUsername, "TELEMT_UNAVAILABLE"); err != nil {
		t.Fatal(err)
	}
	if _, created, err := proxyprovision.Prepare(ctx, db, inviter.User.ProxyUsername, [32]byte{1}, now); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("proxyprovision.Prepare() created = false")
	}
	expires := now.Add(6 * time.Hour)
	if _, err := credit.GrantBucket(ctx, db, inviter.User.ProxyUsername, credit.Grant{
		OriginalBytes: 1234, StartsAt: now.Add(-time.Minute), ExpiresAt: &expires, RewardType: "manual", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := referral.AttributeNewInvitee(ctx, db, invitee, code.Value, now); err != nil || result.Outcome != referral.OutcomeCreated {
		t.Fatalf("create attribution = %#v, %v", result, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE proxy_users SET username = ? WHERE id = ?`, `<unsafe&name>`, inviter.User.ProxyUserID); err != nil {
		t.Fatal(err)
	}

	beforeTelegram := countHTTPRows(t, db, "telegram_users")
	beforeProxy := countHTTPRows(t, db, "proxy_users")
	beforeProvisioning := countHTTPRows(t, db, "proxy_user_provisioning")
	beforeReferrals := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	response := perform(server.Handler(), http.MethodGet, "/users?limit=1", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("users page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		">9302<",
		"&lt;unsafe&amp;name&gt;",
		">false<",
		">error<",
		"<th>Provisioning phase</th>",
		">prepared<",
		">TELEMT_UNAVAILABLE<",
		">1234<",
		">1<",
		expires.Format(time.RFC3339),
		`href="/users?before_id=` + strconv.FormatInt(inviter.User.ID, 10) + `&amp;limit=1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("users page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, `<unsafe&name>`) || strings.Contains(body, ">9301<") || strings.Contains(body, "secret_sha256") {
		t.Fatalf("users page leaked unescaped/older/secret-digest material: %s", body)
	}
	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "proxy_user_provisioning") != beforeProvisioning || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("users page rendering mutated authoritative state")
	}

	older := perform(server.Handler(), http.MethodGet, "/users?before_id="+strconv.FormatInt(inviter.User.ID, 10)+"&limit=1", nil, cookie)
	if older.Code != http.StatusOK || !strings.Contains(older.Body.String(), ">9301<") || !strings.Contains(older.Body.String(), ">not provisioned<") || strings.Contains(older.Body.String(), ">9302<") {
		t.Fatalf("older users page pagination/provisioning mismatch: %d %s", older.Code, older.Body.String())
	}
}

func TestUserPageRendersEstablishedLifecycleControlsAndCSRF(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	resolved, err := telegramuser.Resolve(ctx, db, 9401, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	response := perform(server.Handler(), http.MethodGet, "/users", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("users lifecycle page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`data-csrf="` + sessionCSRF(cookie[0].Value) + `"`,
		`data-proxy-username="` + resolved.User.ProxyUsername + `"`,
		`data-proxy-action="disable"`,
		`data-proxy-action="rotate-secret"`,
		`data-proxy-action="reconcile"`,
		`"X-CSRF-Token": csrf`,
		`"/api/proxy/users/" + encodeURIComponent(username)`,
		`window.location.reload()`,
		`Quota reconciliation queued.`,
		`Shown once. Save it now; it will not be shown again.`,
		`value.textContent = secret`,
		`.slice(0, 256)`,
		`>not provisioned<`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("users lifecycle page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, `data-proxy-action="enable"`) {
		t.Fatalf("enabled row rendered enable action: %s", body)
	}
	if strings.Contains(body, proxyUserTestSecret) || strings.Contains(body, "navigator.clipboard") || strings.Contains(body, "secret_sha256") {
		t.Fatalf("users page leaked/copied secret material: %s", body)
	}

	if _, err := proxyuser.SetDesiredEnabled(ctx, db, resolved.User.ProxyUsername, false); err != nil {
		t.Fatal(err)
	}
	disabled := perform(server.Handler(), http.MethodGet, "/users", nil, cookie)
	if disabled.Code != http.StatusOK || !strings.Contains(disabled.Body.String(), `data-proxy-action="enable"`) || strings.Contains(disabled.Body.String(), `data-proxy-action="disable"`) {
		t.Fatalf("disabled user actions mismatch: %d %s", disabled.Code, disabled.Body.String())
	}
}

func TestUserPageDoesNotGuessLifecycleTargetWithoutProxyUsername(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	resolved, err := telegramuser.Resolve(ctx, db, 9402, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE proxy_users SET username = '' WHERE id = ?`, resolved.User.ProxyUserID); err != nil {
		t.Fatal(err)
	}

	response := perform(server.Handler(), http.MethodGet, "/users", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("users empty proxy username page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `data-proxy-actions-unavailable`) || !strings.Contains(body, `data-proxy-username=""`) {
		t.Fatalf("empty proxy username did not render safe unavailable state: %s", body)
	}
	for _, action := range []string{"enable", "disable", "rotate-secret", "reconcile"} {
		if strings.Contains(body, `data-proxy-action="`+action+`"`) {
			t.Fatalf("empty proxy username rendered %q lifecycle target: %s", action, body)
		}
	}
}

func TestUserPageRejectsInvalidPaginationWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	beforeTelegram := countHTTPRows(t, db, "telegram_users")
	beforeProxy := countHTTPRows(t, db, "proxy_users")
	beforeProvisioning := countHTTPRows(t, db, "proxy_user_provisioning")
	beforeReferrals := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	for _, path := range []string{
		"/users?before_id=0",
		"/users?limit=0",
		"/users?unknown=1",
	} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"USER_INVENTORY_INVALID"`) {
			t.Fatalf("invalid users page query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "proxy_user_provisioning") != beforeProvisioning || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("invalid users page query mutated authoritative state")
	}
}

func TestUserPageEmptyStateAndDashboardLink(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)

	users := perform(server.Handler(), http.MethodGet, "/users", nil, cookie)
	if users.Code != http.StatusOK || !strings.Contains(users.Body.String(), "No Telegram users yet.") {
		t.Fatalf("empty users page = %d %s", users.Code, users.Body.String())
	}
	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, cookie)
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), `href="/users"`) {
		t.Fatalf("dashboard users link = %d %s", dashboard.Code, dashboard.Body.String())
	}
}

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
	for _, unwanted := range []string{">9351<", ">9352<", ">9354<", "secret_sha256"} {
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

	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "proxy_user_provisioning") != beforeProvisioning || countHTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("provisioning phase page filter mutated authoritative state")
	}

	_ = unprovisioned
}
