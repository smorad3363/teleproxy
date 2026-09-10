package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TPROXY_HTTP_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
}

func TestLoadRejectsInvalidAddress(t *testing.T) {
	t.Setenv("TPROXY_HTTP_ADDR", "not-an-address")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadAcceptsExplicitAddress(t *testing.T) {
	t.Setenv("TPROXY_HTTP_ADDR", "0.0.0.0:9000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != "0.0.0.0:9000" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
}
