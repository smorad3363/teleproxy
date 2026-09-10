package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr             = "127.0.0.1:8080"
	defaultDatabasePath         = "data/teleproxy.db"
	defaultAdminUser            = "admin"
	defaultReconcileConcurrency = 2
)

type Config struct {
	HTTPAddr              string
	DatabasePath          string
	BootstrapAdminUser    string
	BootstrapPasswordFile string
	CookieSecure          bool
	TelemtAPIURL          string
	TelemtAPITokenFile    string
	ReconcileConcurrency  int
	BotTokenFile          string
	BotWebhookSecretFile  string
	BotUsername           string
	ReadHeaderTimeout     time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
}

func Load() (Config, error) {
	cookieSecure, err := envBool("TPROXY_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	reconcileConcurrency, err := envIntRange("TPROXY_RECONCILE_CONCURRENCY", defaultReconcileConcurrency, 1, 32)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:              envOrDefault("TPROXY_HTTP_ADDR", defaultHTTPAddr),
		DatabasePath:          envOrDefault("TPROXY_DATABASE_PATH", defaultDatabasePath),
		BootstrapAdminUser:    envOrDefault("TPROXY_BOOTSTRAP_ADMIN_USER", defaultAdminUser),
		BootstrapPasswordFile: os.Getenv("TPROXY_BOOTSTRAP_PASSWORD_FILE"),
		CookieSecure:          cookieSecure,
		TelemtAPIURL:          os.Getenv("TPROXY_TELEMT_API_URL"),
		TelemtAPITokenFile:    os.Getenv("TPROXY_TELEMT_API_TOKEN_FILE"),
		ReconcileConcurrency:  reconcileConcurrency,
		BotTokenFile:          os.Getenv("TPROXY_BOT_TOKEN_FILE"),
		BotWebhookSecretFile:  os.Getenv("TPROXY_BOT_WEBHOOK_SECRET_FILE"),
		BotUsername:           os.Getenv("TPROXY_BOT_USERNAME"),
		ReadHeaderTimeout:     5 * time.Second,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
	}

	if err := validateAddr(cfg.HTTPAddr); err != nil {
		return Config{}, fmt.Errorf("TPROXY_HTTP_ADDR: %w", err)
	}
	if cfg.DatabasePath == "" {
		return Config{}, fmt.Errorf("TPROXY_DATABASE_PATH must not be empty")
	}
	if cfg.BootstrapAdminUser == "" {
		return Config{}, fmt.Errorf("TPROXY_BOOTSTRAP_ADMIN_USER must not be empty")
	}
	if (cfg.TelemtAPIURL == "") != (cfg.TelemtAPITokenFile == "") {
		return Config{}, fmt.Errorf("TPROXY_TELEMT_API_URL and TPROXY_TELEMT_API_TOKEN_FILE must be configured together")
	}
	botFields := 0
	for _, value := range []string{cfg.BotTokenFile, cfg.BotWebhookSecretFile, cfg.BotUsername} {
		if value != "" {
			botFields++
		}
	}
	if botFields != 0 && botFields != 3 {
		return Config{}, fmt.Errorf("TPROXY_BOT_TOKEN_FILE, TPROXY_BOT_WEBHOOK_SECRET_FILE and TPROXY_BOT_USERNAME must be configured together")
	}
	if botFields == 3 {
		if err := validateBotUsername(cfg.BotUsername); err != nil {
			return Config{}, fmt.Errorf("TPROXY_BOT_USERNAME: %w", err)
		}
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return parsed, nil
}

func envIntRange(key string, fallback, minimum, maximum int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", key, minimum, maximum)
	}
	return parsed, nil
}

func validateAddr(addr string) error {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("must be host:port: %w", err)
	}

	if host != "" && net.ParseIP(host) == nil && host != "localhost" {
		return fmt.Errorf("host must be an IP address or localhost")
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func validateBotUsername(value string) error {
	value = strings.TrimPrefix(value, "@")
	if len(value) < 1 || len(value) > 64 {
		return fmt.Errorf("must contain between 1 and 64 characters")
	}
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return fmt.Errorf("contains unsupported characters")
	}
	return nil
}
