package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
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
	if strings.Contains(body, `<unsafe&name>`) || strings.Contains(body, ">9301<") {
		t.Fatalf("users page leaked unescaped/older row: %s", body)
	}
	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("users page rendering mutated authoritative state")
	}

	older := perform(server.Handler(), http.MethodGet, "/users?before_id="+strconv.FormatInt(inviter.User.ID, 10)+"&limit=1", nil, cookie)
	if older.Code != http.StatusOK || !strings.Contains(older.Body.String(), ">9301<") || strings.Contains(older.Body.String(), ">9302<") {
		t.Fatalf("older users page pagination mismatch: %d %s", older.Code, older.Body.String())
	}
}

func TestUserPageRejectsInvalidPaginationWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	beforeTelegram := countHTTPRows(t, db, "telegram_users")
	beforeProxy := countHTTPRows(t, db, "proxy_users")
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
	if countHTTPRows(t, db, "telegram_users") != beforeTelegram || countHTTPRows(t, db, "proxy_users") != beforeProxy || countHTTPRows(t, db, "referral_attributions") != beforeReferrals || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
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
