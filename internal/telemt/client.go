package telemt

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxHealthBodyBytes = 64 << 10

type State string

const (
	StateHealthy         State = "healthy"
	StateUnauthorized    State = "unauthorized"
	StateUnavailable     State = "unavailable"
	StateInvalidResponse State = "invalid_response"
	StateNotConfigured   State = "not_configured"
)

type Health struct {
	State    State `json:"state"`
	ReadOnly bool  `json:"read_only"`
}

type Checker interface {
	Health(context.Context) Health
}

type Client struct {
	healthURL     string
	authorization string
	httpClient    *http.Client
}

func NewFromTokenFile(baseURL, tokenFile string, timeout time.Duration) (*Client, error) {
	if tokenFile == "" {
		return nil, fmt.Errorf("Telemt API token file is required")
	}
	info, err := os.Stat(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("stat Telemt API token file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Telemt API token file must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("Telemt API token file must not be accessible by group or others")
	}
	if info.Size() > 4096 {
		return nil, fmt.Errorf("Telemt API token file is too large")
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("read Telemt API token file: %w", err)
	}
	return New(baseURL, strings.TrimSpace(string(data)), timeout)
}

func New(baseURL, token string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("Telemt API URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("Telemt API URL scheme must be http or https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("Telemt API URL must contain only scheme and host")
	}
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return nil, fmt.Errorf("Telemt API token must be 32 bytes encoded as hexadecimal")
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	parsed.Path = "/v1/health"
	return &Client{
		healthURL:     parsed.String(),
		authorization: "Bearer " + token,
		httpClient:    &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) Health(ctx context.Context) Health {
	if c == nil {
		return Health{State: StateNotConfigured}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.healthURL, nil)
	if err != nil {
		return Health{State: StateInvalidResponse}
	}
	req.Header.Set("Authorization", c.authorization)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Health{State: StateUnavailable}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return Health{State: StateUnauthorized}
	}
	if resp.StatusCode != http.StatusOK {
		return Health{State: StateUnavailable}
	}

	var payload struct {
		OK   bool `json:"ok"`
		Data struct {
			Status   string `json:"status"`
			ReadOnly bool   `json:"read_only"`
		} `json:"data"`
	}
	reader := io.LimitReader(resp.Body, maxHealthBodyBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil || len(data) > maxHealthBodyBytes || json.Unmarshal(data, &payload) != nil {
		return Health{State: StateInvalidResponse}
	}
	if !payload.OK || payload.Data.Status != "ok" {
		return Health{State: StateInvalidResponse}
	}
	return Health{State: StateHealthy, ReadOnly: payload.Data.ReadOnly}
}
