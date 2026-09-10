package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const defaultHTTPAddr = "127.0.0.1:8080"

type Config struct {
	HTTPAddr          string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:          envOrDefault("TPROXY_HTTP_ADDR", defaultHTTPAddr),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := validateAddr(cfg.HTTPAddr); err != nil {
		return Config{}, fmt.Errorf("TPROXY_HTTP_ADDR: %w", err)
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
