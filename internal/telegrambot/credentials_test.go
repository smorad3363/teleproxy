package telegrambot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreTokenFileAndConfigured(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bot-token")
	configured, err := TokenFileConfigured(path)
	if err != nil || configured {
		t.Fatalf("missing TokenFileConfigured = %v, %v", configured, err)
	}
	const token = "123456:Abc_def-XYZ"
	if err := StoreTokenFile(path, token); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token mode = %o, want 600", got)
	}
	configured, err = TokenFileConfigured(path)
	if err != nil || !configured {
		t.Fatalf("configured TokenFileConfigured = %v, %v", configured, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != token {
		t.Fatal("stored token mismatch")
	}
}

func TestStoreTokenFileRejectsInvalidWithoutReplacing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bot-token")
	const token = "123456:Abc_def-XYZ"
	if err := StoreTokenFile(path, token); err != nil {
		t.Fatal(err)
	}
	if err := StoreTokenFile(path, "not-a-token"); err == nil {
		t.Fatal("StoreTokenFile accepted invalid token")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != token {
		t.Fatal("invalid replacement changed token")
	}
}

func TestEnsureWebhookSecretFileCreatesAndPreservesSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "webhook-secret")
	if err := EnsureWebhookSecretFile(path); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("secret mode = %o, want 600", got)
	}
	if _, err := LoadWebhookSecretFile(path); err != nil {
		t.Fatalf("generated secret invalid: %v", err)
	}
	if err := EnsureWebhookSecretFile(path); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("EnsureWebhookSecretFile replaced existing valid secret")
	}
}
