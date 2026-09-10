package telemt

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxResponseBodyBytes = 1 << 20

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

type User struct {
	Username  string `json:"username"`
	Enabled   bool   `json:"enabled"`
	InRuntime bool   `json:"in_runtime"`
}

type Credential struct {
	User   User   `json:"user"`
	Secret string `json:"secret"`
}

type FailureCode string

const (
	FailureUnauthorized  FailureCode = "unauthorized"
	FailureForbidden     FailureCode = "forbidden"
	FailureNotFound      FailureCode = "not_found"
	FailureConflict      FailureCode = "conflict"
	FailureRejected      FailureCode = "rejected"
	FailureUnavailable   FailureCode = "unavailable"
	FailureInvalidOutput FailureCode = "invalid_response"
)

type APIError struct {
	Code FailureCode
}

func (e *APIError) Error() string {
	return "Telemt request failed: " + string(e.Code)
}

func FailureCodeOf(err error) FailureCode {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

type Client struct {
	baseURL       string
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
	parsed.Path = ""
	return &Client{
		baseURL:       strings.TrimRight(parsed.String(), "/"),
		authorization: "Bearer " + token,
		httpClient:    &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) Health(ctx context.Context) Health {
	if c == nil {
		return Health{State: StateNotConfigured}
	}
	var data struct {
		Status   string `json:"status"`
		ReadOnly bool   `json:"read_only"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/health", nil, []int{http.StatusOK}, &data); err != nil {
		switch FailureCodeOf(err) {
		case FailureUnauthorized, FailureForbidden:
			return Health{State: StateUnauthorized}
		case FailureInvalidOutput:
			return Health{State: StateInvalidResponse}
		default:
			return Health{State: StateUnavailable}
		}
	}
	if data.Status != "ok" {
		return Health{State: StateInvalidResponse}
	}
	return Health{State: StateHealthy, ReadOnly: data.ReadOnly}
}

func (c *Client) CreateUser(ctx context.Context, username string, enabled bool) (Credential, error) {
	if err := validateUsername(username); err != nil {
		return Credential{}, err
	}
	body := struct {
		Username string `json:"username"`
		Enabled  bool   `json:"enabled"`
	}{Username: username, Enabled: enabled}
	var credential Credential
	if err := c.doJSON(ctx, http.MethodPost, "/v1/users", body, []int{http.StatusCreated, http.StatusAccepted}, &credential); err != nil {
		return Credential{}, err
	}
	if err := validateCredential(credential, username); err != nil {
		return Credential{}, err
	}
	return credential, nil
}

func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	if err := c.doJSON(ctx, http.MethodGet, "/v1/users", nil, []int{http.StatusOK}, &users); err != nil {
		return nil, err
	}
	for _, user := range users {
		if validateUsername(user.Username) != nil {
			return nil, &APIError{Code: FailureInvalidOutput}
		}
	}
	return users, nil
}

func (c *Client) SetUserEnabled(ctx context.Context, username string, enabled bool) (User, error) {
	if err := validateUsername(username); err != nil {
		return User{}, err
	}
	action := "disable"
	if enabled {
		action = "enable"
	}
	var user User
	path := "/v1/users/" + url.PathEscape(username) + "/" + action
	if err := c.doJSON(ctx, http.MethodPost, path, struct{}{}, []int{http.StatusOK, http.StatusAccepted}, &user); err != nil {
		return User{}, err
	}
	if user.Username != username || user.Enabled != enabled {
		return User{}, &APIError{Code: FailureInvalidOutput}
	}
	return user, nil
}

func (c *Client) RotateUserSecret(ctx context.Context, username string) (Credential, error) {
	if err := validateUsername(username); err != nil {
		return Credential{}, err
	}
	var credential Credential
	path := "/v1/users/" + url.PathEscape(username) + "/rotate-secret"
	if err := c.doJSON(ctx, http.MethodPost, path, struct{}{}, []int{http.StatusOK, http.StatusAccepted}, &credential); err != nil {
		return Credential{}, err
	}
	if err := validateCredential(credential, username); err != nil {
		return Credential{}, err
	}
	return credential, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestBody any, accepted []int, output any) error {
	if c == nil {
		return &APIError{Code: FailureUnavailable}
	}
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return &APIError{Code: FailureInvalidOutput}
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return &APIError{Code: FailureInvalidOutput}
	}
	req.Header.Set("Authorization", c.authorization)
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &APIError{Code: FailureUnavailable}
	}
	defer resp.Body.Close()

	if !containsStatus(accepted, resp.StatusCode) {
		return &APIError{Code: classifyHTTPStatus(resp.StatusCode)}
	}
	reader := io.LimitReader(resp.Body, maxResponseBodyBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil || len(data) > maxResponseBodyBytes {
		return &APIError{Code: FailureInvalidOutput}
	}
	var envelope struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || !envelope.OK || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return &APIError{Code: FailureInvalidOutput}
	}
	if json.Unmarshal(envelope.Data, output) != nil {
		return &APIError{Code: FailureInvalidOutput}
	}
	return nil
}

func classifyHTTPStatus(status int) FailureCode {
	switch status {
	case http.StatusUnauthorized:
		return FailureUnauthorized
	case http.StatusForbidden:
		return FailureForbidden
	case http.StatusNotFound:
		return FailureNotFound
	case http.StatusConflict:
		return FailureConflict
	case http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType:
		return FailureRejected
	default:
		if status >= 500 {
			return FailureUnavailable
		}
		return FailureInvalidOutput
	}
}

func containsStatus(accepted []int, status int) bool {
	for _, candidate := range accepted {
		if candidate == status {
			return true
		}
	}
	return false
}

func validateCredential(credential Credential, username string) error {
	if credential.User.Username != username || !validHex(credential.Secret, 16) {
		return &APIError{Code: FailureInvalidOutput}
	}
	return nil
}

func validateUsername(username string) error {
	if len(username) < 1 || len(username) > 64 {
		return fmt.Errorf("Telemt username must be between 1 and 64 characters")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return fmt.Errorf("Telemt username contains unsupported characters")
	}
	return nil
}

func validHex(value string, bytesLen int) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == bytesLen
}
