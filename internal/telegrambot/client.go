package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const maxResponseBodyBytes = 1 << 20

type FailureCode string

const (
	FailureUnauthorized  FailureCode = "unauthorized"
	FailureRateLimited   FailureCode = "rate_limited"
	FailureRejected      FailureCode = "rejected"
	FailureUnavailable   FailureCode = "unavailable"
	FailureInvalidOutput FailureCode = "invalid_response"
)

type APIError struct {
	Code FailureCode
}

func (e *APIError) Error() string {
	return "Telegram Bot API request failed: " + string(e.Code)
}

func FailureCodeOf(err error) FailureCode {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type SentMessage struct {
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

func NewFromTokenFile(tokenFile string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(tokenFile) == "" {
		return nil, fmt.Errorf("Telegram Bot token file is required")
	}
	info, err := os.Stat(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("stat Telegram Bot token file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Telegram Bot token file must be a regular file")
	}
	perm := info.Mode().Perm()
	if perm&0o400 == 0 {
		return nil, fmt.Errorf("Telegram Bot token file must be owner-readable")
	}
	if perm&0o077 != 0 {
		return nil, fmt.Errorf("Telegram Bot token file must not be accessible by group or others")
	}
	if info.Size() <= 0 || info.Size() > 4096 {
		return nil, fmt.Errorf("Telegram Bot token file size is invalid")
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("read Telegram Bot token file: %w", err)
	}
	return New(strings.TrimSpace(string(data)), timeout)
}

func New(token string, timeout time.Duration) (*Client, error) {
	return newClient("https://api.telegram.org", token, timeout)
}

func newClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("Telegram Bot API URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("Telegram Bot API URL scheme must be http or https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("Telegram Bot API URL must contain only scheme and host")
	}
	if err := validateToken(token); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	parsed.Path = ""
	return &Client{
		baseURL:    strings.TrimRight(parsed.String(), "/"),
		token:      token,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) (SentMessage, error) {
	if c == nil {
		return SentMessage{}, &APIError{Code: FailureUnavailable}
	}
	if chatID == 0 {
		return SentMessage{}, fmt.Errorf("Telegram chat ID must not be zero")
	}
	if !utf8.ValidString(text) {
		return SentMessage{}, fmt.Errorf("Telegram message text must be valid UTF-8")
	}
	characters := utf8.RuneCountInString(text)
	if characters < 1 || characters > 4096 {
		return SentMessage{}, fmt.Errorf("Telegram message text must contain between 1 and 4096 characters")
	}

	payload, err := json.Marshal(struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}{ChatID: chatID, Text: text})
	if err != nil {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+c.token+"/sendMessage", bytes.NewReader(payload))
	if err != nil {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Never surface net/http's URL-bearing error: Bot API URLs contain the token.
		return SentMessage{}, &APIError{Code: FailureUnavailable}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SentMessage{}, &APIError{Code: classifyFailure(resp.StatusCode, 0)}
	}

	reader := io.LimitReader(resp.Body, maxResponseBodyBytes+1)
	body, err := io.ReadAll(reader)
	if err != nil || len(body) > maxResponseBodyBytes {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	var envelope struct {
		OK        bool            `json:"ok"`
		Result    json.RawMessage `json:"result"`
		ErrorCode int             `json:"error_code"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	if !envelope.OK {
		return SentMessage{}, &APIError{Code: classifyFailure(resp.StatusCode, envelope.ErrorCode)}
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	var message SentMessage
	if json.Unmarshal(envelope.Result, &message) != nil || message.MessageID <= 0 || message.Chat.ID != chatID {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	return message, nil
}

func validateToken(token string) error {
	if token == "" || token != strings.TrimSpace(token) || len(token) > 256 {
		return fmt.Errorf("Telegram Bot token format is invalid")
	}
	botID, secret, ok := strings.Cut(token, ":")
	if !ok || botID == "" || secret == "" {
		return fmt.Errorf("Telegram Bot token format is invalid")
	}
	for _, ch := range botID {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("Telegram Bot token format is invalid")
		}
	}
	for _, ch := range secret {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return fmt.Errorf("Telegram Bot token format is invalid")
	}
	return nil
}

func classifyFailure(status, telegramCode int) FailureCode {
	code := telegramCode
	if code == 0 {
		code = status
	}
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden:
		return FailureUnauthorized
	case http.StatusTooManyRequests:
		return FailureRateLimited
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict:
		return FailureRejected
	default:
		if code >= 500 || status >= 500 {
			return FailureUnavailable
		}
		return FailureInvalidOutput
	}
}
