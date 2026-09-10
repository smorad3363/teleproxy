package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"TPROXY_HTTP_ADDR",
		"TPROXY_DATABASE_PATH",
		"TPROXY_BOOTSTRAP_ADMIN_USER",
		"TPROXY_BOOTSTRAP_PASSWORD_FILE",
		"TPROXY_COOKIE_SECURE",
		"TPROXY_TELEMT_API_URL",
		"TPROXY_TELEMT_API_TOKEN_FILE",
		"TPROXY_RECONCILE_CONCURRENCY",
		"TPROXY_BOT_TOKEN_FILE",
		"TPROXY_BOT_WEBHOOK_SECRET_FILE",
		"TPROXY_BOT_USERNAME",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
	if cfg.DatabasePath != defaultDatabasePath {
		t.Fatalf("DatabasePath = %q", cfg.DatabasePath)
	}
	if cfg.BootstrapAdminUser != defaultAdminUser {
		t.Fatalf("BootstrapAdminUser = %q", cfg.BootstrapAdminUser)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure = true, want false by default")
	}
	if cfg.TelemtAPIURL != "" || cfg.TelemtAPITokenFile != "" {
		t.Fatalf("Telemt client unexpectedly configured: %#v", cfg)
	}
	if cfg.ReconcileConcurrency != defaultReconcileConcurrency {
		t.Fatalf("ReconcileConcurrency = %d, want %d", cfg.ReconcileConcurrency, defaultReconcileConcurrency)
	}
	if cfg.BotTokenFile != "" || cfg.BotWebhookSecretFile != "" || cfg.BotUsername != "" {
		t.Fatalf("Telegram Bot unexpectedly configured: %#v", cfg)
	}
}

func TestLoadRejectsInvalidAddress(t *testing.T) {
	t.Setenv("TPROXY_HTTP_ADDR", "not-an-address")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadAcceptsExplicitSettings(t *testing.T) {
	t.Setenv("TPROXY_HTTP_ADDR", "0.0.0.0:9000")
	t.Setenv("TPROXY_DATABASE_PATH", "/var/lib/teleproxy/teleproxy.db")
	t.Setenv("TPROXY_BOOTSTRAP_ADMIN_USER", "rootadmin")
	t.Setenv("TPROXY_BOOTSTRAP_PASSWORD_FILE", "/run/secrets/admin-password")
	t.Setenv("TPROXY_COOKIE_SECURE", "true")
	t.Setenv("TPROXY_TELEMT_API_URL", "http://telemt:9091")
	t.Setenv("TPROXY_TELEMT_API_TOKEN_FILE", "/run/secrets/telemt-api-token")
	t.Setenv("TPROXY_RECONCILE_CONCURRENCY", "4")
	t.Setenv("TPROXY_BOT_TOKEN_FILE", "/run/secrets/telegram-bot-token")
	t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", "/run/secrets/telegram-webhook-secret")
	t.Setenv("TPROXY_BOT_USERNAME", "TeleproxyBot")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != "0.0.0.0:9000" || !cfg.CookieSecure {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.TelemtAPIURL != "http://telemt:9091" || cfg.TelemtAPITokenFile != "/run/secrets/telemt-api-token" {
		t.Fatalf("unexpected Telemt config: %#v", cfg)
	}
	if cfg.ReconcileConcurrency != 4 {
		t.Fatalf("ReconcileConcurrency = %d, want 4", cfg.ReconcileConcurrency)
	}
	if cfg.BotTokenFile != "/run/secrets/telegram-bot-token" || cfg.BotWebhookSecretFile != "/run/secrets/telegram-webhook-secret" || cfg.BotUsername != "TeleproxyBot" {
		t.Fatalf("unexpected Telegram Bot config: %#v", cfg)
	}
}

func TestLoadRejectsPartialTelemtConfig(t *testing.T) {
	t.Setenv("TPROXY_TELEMT_API_URL", "http://telemt:9091")
	t.Setenv("TPROXY_TELEMT_API_TOKEN_FILE", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted Telemt URL without token file")
	}
}

func TestLoadRejectsPartialBotConfig(t *testing.T) {
	cases := []struct {
		name     string
		token    string
		secret   string
		username string
	}{
		{name: "token only", token: "/run/token"},
		{name: "secret only", secret: "/run/secret"},
		{name: "username only", username: "TeleproxyBot"},
		{name: "missing username", token: "/run/token", secret: "/run/secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TPROXY_BOT_TOKEN_FILE", tc.token)
			t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", tc.secret)
			t.Setenv("TPROXY_BOT_USERNAME", tc.username)
			if _, err := Load(); err == nil {
				t.Fatal("Load() accepted partial Telegram Bot config")
			}
		})
	}
}

func TestLoadRejectsInvalidBotUsername(t *testing.T) {
	t.Setenv("TPROXY_BOT_TOKEN_FILE", "/run/token")
	t.Setenv("TPROXY_BOT_WEBHOOK_SECRET_FILE", "/run/secret")
	for _, username := range []string{"bad-name", "with space", "@"} {
		t.Run(username, func(t *testing.T) {
			t.Setenv("TPROXY_BOT_USERNAME", username)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted bot username %q", username)
			}
		})
	}
}

func TestLoadRejectsInvalidCookieSecure(t *testing.T) {
	t.Setenv("TPROXY_COOKIE_SECURE", "maybe")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want boolean validation error")
	}
}

func TestLoadRejectsInvalidReconcileConcurrency(t *testing.T) {
	for _, value := range []string{"0", "33", "many"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("TPROXY_RECONCILE_CONCURRENCY", value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted concurrency %q", value)
			}
		})
	}
}
