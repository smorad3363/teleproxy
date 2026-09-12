package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestBotRuntimeSettingsAPIRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	unauthorized := botSettingsRequest(server.Handler(), http.MethodGet, "/api/settings/bot", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), "\"code\":\"AUTH_REQUIRED\"") {
		t.Fatalf("unauthorized GET = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingCSRF := botSettingsRequest(server.Handler(), http.MethodPut, "/api/settings/bot", "{\"username\":\"TeleproxyBot\",\"admin_chat_id\":123,\"enabled\":false,\"token\":\"\"}", cookie, "")
	if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), "\"code\":\"CSRF_INVALID\"") {
		t.Fatalf("missing CSRF = %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestBotRuntimeSettingsAPIStoresSecretOutsideDatabase(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	secretPath := filepath.Join(dir, "webhook-secret")
	t.Setenv("TPROXY_BOT_TOKEN_FILE", tokenPath)
	t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", secretPath)

	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	const token = "123456:Abc_def-XYZ"
	body := "{\"username\":\"@TeleproxyBot\",\"admin_chat_id\":987654321,\"enabled\":true,\"token\":\"" + token + "\"}"
	response := botSettingsRequest(server.Handler(), http.MethodPut, "/api/settings/bot", body, cookie, csrf)
	if response.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), token) || !strings.Contains(response.Body.String(), "\"token_configured\":true") || !strings.Contains(response.Body.String(), "\"restart_required\":true") {
		t.Fatalf("PUT leaked token or missed status: %s", response.Body.String())
	}
	stored, configured, err := settings.BotRuntime(context.Background(), db)
	if err != nil || !configured {
		t.Fatalf("BotRuntime = %#v configured=%v err=%v", stored, configured, err)
	}
	if stored.Username != "TeleproxyBot" || stored.AdminChatID != 987654321 || !stored.Enabled {
		t.Fatalf("stored settings = %#v", stored)
	}
	for _, path := range []string{tokenPath, secretPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("%s mode = %o, want 600", filepath.Base(path), got)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bot_runtime_settings WHERE username LIKE '%' || ? || '%'`, token).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("Bot token was persisted in SQLite")
	}

	get := botSettingsRequest(server.Handler(), http.MethodGet, "/api/settings/bot", "", cookie, "")
	if get.Code != http.StatusOK || strings.Contains(get.Body.String(), token) || !strings.Contains(get.Body.String(), "\"admin_chat_id\":987654321") {
		t.Fatalf("GET = %d %s", get.Code, get.Body.String())
	}
}

func TestBotRuntimeSettingsRequiresTokenBeforeEnable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TPROXY_BOT_TOKEN_FILE", filepath.Join(dir, "token"))
	t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", filepath.Join(dir, "webhook-secret"))
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)

	response := botSettingsRequest(server.Handler(), http.MethodPut, "/api/settings/bot", "{\"username\":\"TeleproxyBot\",\"admin_chat_id\":123,\"enabled\":true,\"token\":\"\"}", cookie, csrf)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "\"code\":\"BOT_TOKEN_REQUIRED\"") {
		t.Fatalf("enable without token = %d %s", response.Code, response.Body.String())
	}
	if _, configured, err := settings.BotRuntime(context.Background(), db); err != nil || configured {
		t.Fatalf("failed enable mutated settings: configured=%v err=%v", configured, err)
	}
}

func TestSettingsPageRendersBotSurfaceWithoutToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	secretPath := filepath.Join(dir, "webhook-secret")
	t.Setenv("TPROXY_BOT_TOKEN_FILE", tokenPath)
	t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", secretPath)
	if err := os.WriteFile(tokenPath, []byte("123456:Abc_def-XYZ\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)
	if _, err := settings.SetBotRuntime(context.Background(), db, settings.BotRuntimeSettings{Username: "TeleproxyBot", AdminChatID: 9007199254740993, Enabled: true}, time.Unix(1_700_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	response := perform(server.Handler(), http.MethodGet, "/settings", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("settings page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`content="` + csrf + `"`,
		`name="username" type="text" maxlength="64" value="TeleproxyBot"`,
		`name="admin_chat_id" type="text" inputmode="numeric" pattern="[1-9][0-9]*" value="9007199254740993"`,
		`name="token" type="password"`,
		`fetch('/api/settings/bot'`,
		`fetch('/api/settings/bot/test'`,
		`tproxy restart`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "123456:Abc_def-XYZ") || strings.Contains(body, "Number(adminChatID)") || strings.Contains(body, "parseInt(adminChatID") {
		t.Fatalf("settings page leaked token or uses lossy chat ID conversion: %s", body)
	}
}

func TestBotRuntimeSettingsTestRequiresSavedConfiguration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TPROXY_BOT_TOKEN_FILE", filepath.Join(dir, "token"))
	t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", filepath.Join(dir, "webhook-secret"))
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	response := botSettingsRequest(server.Handler(), http.MethodPost, "/api/settings/bot/test", "", cookie, csrf)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "\"code\":\"BOT_NOT_CONFIGURED\"") {
		t.Fatalf("test before save = %d %s", response.Code, response.Body.String())
	}
}

func botSettingsRequest(handler http.Handler, method, path, body string, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
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
