package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/settings"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestStartGiftSettingsAPIRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/settings/start-gift", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized GET = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingCSRF := performJSON(server.Handler(), http.MethodPut, "/api/settings/start-gift", `{"bytes":250000000}`, cookie, "")
	if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("missing CSRF = %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestStartGiftSettingsAPIGetDefaultAndUpdate(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)

	defaults := performJSON(server.Handler(), http.MethodGet, "/api/settings/start-gift", "", cookie, "")
	if defaults.Code != http.StatusOK || !strings.Contains(defaults.Body.String(), `"bytes":100000000`) {
		t.Fatalf("default GET = %d %s", defaults.Code, defaults.Body.String())
	}
	if got := defaults.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("default GET Cache-Control = %q", got)
	}

	updated := performJSON(server.Handler(), http.MethodPut, "/api/settings/start-gift", `{"bytes":350000000}`, cookie, csrf)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"bytes":350000000`) {
		t.Fatalf("PUT = %d %s", updated.Code, updated.Body.String())
	}
	stored, err := settings.StartGiftBytes(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 350_000_000 {
		t.Fatalf("stored start gift = %d", stored)
	}
}

func TestStartGiftSettingsChangeAffectsOnlyFutureExactlyOnceGift(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	now := time.Now().UTC().Truncate(time.Second)

	if err := settings.SetStartGiftBytes(ctx, db, 111_000_000); err != nil {
		t.Fatal(err)
	}
	first, err := telegramuser.Resolve(ctx, db, 9401, now)
	if err != nil {
		t.Fatal(err)
	}
	gift1, err := telegramuser.EnsureStartGift(ctx, db, first.User.TelegramID, now)
	if err != nil || !gift1.Granted || gift1.InitialGiftBytes != 111_000_000 {
		t.Fatalf("first gift = %#v, %v", gift1, err)
	}

	updated := performJSON(server.Handler(), http.MethodPut, "/api/settings/start-gift", `{"bytes":222000000}`, cookie, csrf)
	if updated.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", updated.Code, updated.Body.String())
	}
	var storedOriginal int64
	if err := db.QueryRowContext(ctx, `SELECT original_bytes FROM credit_buckets WHERE proxy_user_id = ? AND reward_type = 'start_gift'`, first.User.ProxyUserID).Scan(&storedOriginal); err != nil {
		t.Fatal(err)
	}
	if storedOriginal != 111_000_000 {
		t.Fatalf("existing start gift mutated to %d", storedOriginal)
	}

	replay, err := telegramuser.EnsureStartGift(ctx, db, first.User.TelegramID, now.Add(time.Minute))
	if err != nil || replay.Granted || replay.InitialGiftBytes != 111_000_000 {
		t.Fatalf("replayed gift = %#v, %v", replay, err)
	}
	second, err := telegramuser.Resolve(ctx, db, 9402, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	gift2, err := telegramuser.EnsureStartGift(ctx, db, second.User.TelegramID, now.Add(2*time.Minute))
	if err != nil || !gift2.Granted || gift2.InitialGiftBytes != 222_000_000 {
		t.Fatalf("second gift = %#v, %v", gift2, err)
	}
	if got := countHTTPRows(t, db, "credit_buckets"); got != 2 {
		t.Fatalf("credit bucket count = %d, want 2", got)
	}
}

func TestStartGiftSettingsAPIRejectsBadRequestsWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	if err := settings.SetStartGiftBytes(ctx, db, 333_000_000); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		body string
		code int
		key  string
	}{
		{name: "malformed", body: `{"bytes":`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "unknown field", body: `{"bytes":250000000,"unknown":1}`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "trailing JSON", body: `{"bytes":250000000}{}`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "zero", body: `{"bytes":0}`, code: http.StatusBadRequest, key: `"code":"START_GIFT_INVALID"`},
		{name: "negative", body: `{"bytes":-1}`, code: http.StatusBadRequest, key: `"code":"START_GIFT_INVALID"`},
		{name: "overflow", body: `{"bytes":9223372036854775808}`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "oversized", body: strings.Repeat(" ", int(maxStartGiftSettingsBodyBytes)+1) + `{}`, code: http.StatusRequestEntityTooLarge, key: `"code":"REQUEST_TOO_LARGE"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performJSON(server.Handler(), http.MethodPut, "/api/settings/start-gift", test.body, cookie, csrf)
			if response.Code != test.code || !strings.Contains(response.Body.String(), test.key) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			stored, err := settings.StartGiftBytes(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			if stored != 333_000_000 {
				t.Fatalf("bad request mutated start gift setting: %d", stored)
			}
		})
	}
}
