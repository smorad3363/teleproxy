package telegrambot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultWebhookBodyBytes       int64 = 64 << 10
	defaultWebhookPerSourceSecond       = 100
	defaultWebhookGlobalSecond          = 200
)

type startUpdateHandler interface {
	Handle(context.Context, Update) (StartResponse, bool, error)
}

type messageSender interface {
	SendMessage(context.Context, int64, string) (SentMessage, error)
}

type WebhookHandler struct {
	secretDigest [32]byte
	start        startUpdateHandler
	sender       messageSender
	maxBodyBytes int64
	limiter      *webhookLimiter
}

type webhookOptions struct {
	MaxBodyBytes    int64
	PerSourceSecond int
	GlobalSecond    int
	Now             func() time.Time
}

func LoadWebhookSecretFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("Telegram webhook secret file is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat Telegram webhook secret file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Telegram webhook secret file must be a regular file")
	}
	perm := info.Mode().Perm()
	if perm&0o400 == 0 {
		return "", fmt.Errorf("Telegram webhook secret file must be owner-readable")
	}
	if perm&0o077 != 0 {
		return "", fmt.Errorf("Telegram webhook secret file must not be accessible by group or others")
	}
	if info.Size() <= 0 || info.Size() > 4096 {
		return "", fmt.Errorf("Telegram webhook secret file size is invalid")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read Telegram webhook secret file: %w", err)
	}
	secret := strings.TrimRight(string(data), "\r\n")
	if err := validateWebhookSecret(secret); err != nil {
		return "", err
	}
	return secret, nil
}

func NewWebhookHandler(secret string, start startUpdateHandler, sender messageSender) (*WebhookHandler, error) {
	return newWebhookHandler(secret, start, sender, webhookOptions{})
}

func newWebhookHandler(secret string, start startUpdateHandler, sender messageSender, options webhookOptions) (*WebhookHandler, error) {
	if err := validateWebhookSecret(secret); err != nil {
		return nil, err
	}
	if start == nil {
		return nil, fmt.Errorf("Telegram start handler is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("Telegram message sender is required")
	}
	if options.MaxBodyBytes <= 0 {
		options.MaxBodyBytes = defaultWebhookBodyBytes
	}
	if options.PerSourceSecond <= 0 {
		options.PerSourceSecond = defaultWebhookPerSourceSecond
	}
	if options.GlobalSecond <= 0 {
		options.GlobalSecond = defaultWebhookGlobalSecond
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &WebhookHandler{
		secretDigest: sha256.Sum256([]byte(secret)),
		start:        start,
		sender:       sender,
		maxBodyBytes: options.MaxBodyBytes,
		limiter:      newWebhookLimiter(options.PerSourceSecond, options.GlobalSecond, options.Now),
	}, nil
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorized(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !h.limiter.Allow(sourceAddress(r.RemoteAddr)) {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, h.maxBodyBytes+1))
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if int64(len(body)) > h.maxBodyBytes {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var update Update
	if err := decoder.Decode(&update); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if update.UpdateID <= 0 {
		http.Error(w, "invalid update", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	response, handled, err := h.start.Handle(r.Context(), update)
	if !handled {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if errors.Is(err, ErrInvalidStartPayload) {
		_, _ = h.sender.SendMessage(r.Context(), update.Message.Chat.ID, "This start link is invalid.")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		http.Error(w, "temporary processing failure", http.StatusServiceUnavailable)
		return
	}

	// Once the idempotent start transaction has committed, an outbound Bot API
	// failure must not roll it back. Returning success also avoids Telegram
	// replaying an ambiguously-sent message and creating duplicate replies.
	_, _ = h.sender.SendMessage(r.Context(), response.ChatID, formatStartResponse(response))
	w.WriteHeader(http.StatusNoContent)
}

func (h *WebhookHandler) authorized(candidate string) bool {
	candidateDigest := sha256.Sum256([]byte(candidate))
	return subtle.ConstantTimeCompare(candidateDigest[:], h.secretDigest[:]) == 1
}

func validateWebhookSecret(secret string) error {
	if len(secret) < 1 || len(secret) > 256 {
		return fmt.Errorf("Telegram webhook secret must contain between 1 and 256 characters")
	}
	for _, ch := range secret {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return fmt.Errorf("Telegram webhook secret format is invalid")
	}
	return nil
}

func formatStartResponse(response StartResponse) string {
	accountState := "ready"
	if response.Created {
		accountState = "created"
	}
	if response.ProxyLink == "" {
		return fmt.Sprintf("Teleproxy account %s.\nUser: %s\nRemaining credit: %d bytes\nProxy link provisioning is pending.", accountState, response.ProxyUsername, response.RemainingBytes)
	}
	proxyState := "synchronizing"
	if response.ProxySyncState == "synced" {
		proxyState = "ready"
	}
	return fmt.Sprintf("Teleproxy account %s.\nUser: %s\nRemaining credit: %d bytes\nProxy status: %s.\nProxy link:\n%s", accountState, response.ProxyUsername, response.RemainingBytes, proxyState, response.ProxyLink)
}

func sourceAddress(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err == nil && host != "" {
		return host
	}
	if remote == "" {
		return "unknown"
	}
	return remote
}

type webhookLimiter struct {
	mu              sync.Mutex
	perSourceLimit  int
	globalLimit     int
	now             func() time.Time
	windowUnix      int64
	globalCount     int
	perSourceCounts map[string]int
}

func newWebhookLimiter(perSourceLimit, globalLimit int, now func() time.Time) *webhookLimiter {
	return &webhookLimiter{
		perSourceLimit:  perSourceLimit,
		globalLimit:     globalLimit,
		now:             now,
		perSourceCounts: make(map[string]int),
	}
}

func (l *webhookLimiter) Allow(source string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	window := l.now().Unix()
	if window != l.windowUnix {
		l.windowUnix = window
		l.globalCount = 0
		clear(l.perSourceCounts)
	}
	if l.globalCount >= l.globalLimit || l.perSourceCounts[source] >= l.perSourceLimit {
		return false
	}
	l.globalCount++
	l.perSourceCounts[source]++
	return true
}
