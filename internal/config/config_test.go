package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"TPROXY_HTTP_ADDR",
		"TPROXY_DATABASE_PATH",
		"TPROXY_BOOTSTRAP_ADMIN_USER",
		"TPROXY_BOOTSTRAP_PASSWORD_FILE",
		"TPROXY_COOKIE_SECURE",
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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != "0.0.0.0:9000" || !cfg.CookieSecure {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoadRejectsInvalidCookieSecure(t *testing.T) {
	t.Setenv("TPROXY_COOKIE_SECURE", "maybe")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want boolean validation error")
	}
}
