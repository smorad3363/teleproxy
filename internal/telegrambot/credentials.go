package telegrambot

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func TokenFileConfigured(path string) (bool, error) {
	if strings.TrimSpace(path) == "" {
		return false, nil
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("stat Telegram Bot token file: %w", err)
	}
	if _, err := NewFromTokenFile(path, time.Second); err != nil {
		return false, err
	}
	return true, nil
}

func StoreTokenFile(path, token string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("Telegram Bot token file path is required")
	}
	if err := validateToken(token); err != nil {
		return err
	}
	return writeSecretFile(path, token)
}

func EnsureWebhookSecretFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("Telegram webhook secret file path is required")
	}
	if _, err := os.Stat(path); err == nil {
		_, err = LoadWebhookSecretFile(path)
		return err
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat Telegram webhook secret file: %w", err)
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("generate Telegram webhook secret")
	}
	secret := hex.EncodeToString(raw)
	if err := validateWebhookSecret(secret); err != nil {
		return fmt.Errorf("generate Telegram webhook secret")
	}
	return writeSecretFile(path, secret)
}

func writeSecretFile(path, value string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat secret directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("secret directory is invalid")
	}

	tmp, err := os.CreateTemp(dir, ".teleproxy-secret-*")
	if err != nil {
		return fmt.Errorf("create secret file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("secure secret file: %w", err)
	}
	if _, err := tmp.WriteString(value + "\n"); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write secret file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync secret file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close secret file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace secret file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("secure secret file: %w", err)
	}
	return nil
}
