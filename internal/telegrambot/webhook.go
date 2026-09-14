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
	maxTelegramMessageRunes             = 4096
)

type startUpdateHandler interface {
	Handle(context.Context, Update) (StartResponse, bool, error)
}

type messageSender interface {
	SendMessage(context.Context, int64, string) (SentMessage, error)
}

type inlineKeyboardSender interface {
	SendMessageWithInlineKeyboard(context.Context, int64, string, InlineKeyboardMarkup) (SentMessage, error)
}

type callbackQueryAnswerer interface {
	AnswerCallbackQuery(context.Context, string, string) error
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
	if update.CallbackQuery != nil {
		handleCallbackResult(r.Context(), h.sender, *update.CallbackQuery, response, err)
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
	sendStartResponse(r.Context(), h.sender, response)
	w.WriteHeader(http.StatusNoContent)
}

func handleCallbackResult(ctx context.Context, sender messageSender, query CallbackQuery, response StartResponse, appErr error) {
	answer := "Membership verified."
	if appErr != nil {
		answer = "Membership check failed. Try again."
	} else if len(response.MissingChannels) > 0 {
		answer = "Join all required channels, then recheck."
	}
	if callbackSender, ok := sender.(callbackQueryAnswerer); ok {
		_ = callbackSender.AnswerCallbackQuery(ctx, query.ID, answer)
	}
	if appErr == nil {
		sendStartResponse(ctx, sender, response)
	}
}

func sendStartResponse(ctx context.Context, sender messageSender, response StartResponse) {
	text := formatStartResponse(response)
	if keyboardSender, ok := sender.(inlineKeyboardSender); ok {
		if len(response.MissingChannels) > 0 {
			_, _ = keyboardSender.SendMessageWithInlineKeyboard(ctx, response.ChatID, text, forcedJoinKeyboard(response.MissingChannels))
			return
		}
		if markup, ok := startActionKeyboard(response); ok {
			_, _ = keyboardSender.SendMessageWithInlineKeyboard(ctx, response.ChatID, text, markup)
			return
		}
	}
	_, _ = sender.SendMessage(ctx, response.ChatID, text)
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
	if len(response.MissingChannels) > 0 {
		return formatMissingChannelsWithPrefix(response.MissingChannels, response.ForcedJoinText)
	}
	accountState := "ready"
	if response.Created {
		accountState = "created"
	}
	sections := make([]string, 0, 5)
	if text := boundedCustomText(response.WelcomeText, 1000); text != "" {
		sections = append(sections, text)
	}
	sections = append(sections, fmt.Sprintf("Teleproxy account %s.\nUser: %s\nRemaining credit: %d bytes", accountState, response.ProxyUsername, response.RemainingBytes))
	if response.ProxyLink == "" {
		sections = append(sections, withOptionalPrefix(response.ProxyText, "Proxy link provisioning is pending."))
	} else {
		proxyState := "synchronizing"
		if response.ProxySyncState == "synced" {
			proxyState = "ready"
		}
		sections = append(sections, withOptionalPrefix(response.ProxyText, fmt.Sprintf("Proxy status: %s.\nProxy link:\n%s", proxyState, response.ProxyLink)))
	}
	if response.ReferralCode != "" {
		referral := fmt.Sprintf("Referral code: %s\nSuccessful referrals: %d", response.ReferralCode, response.ReferralCount)
		if response.ReferralLink != "" {
			referral = fmt.Sprintf("Referral link:\n%s\nSuccessful referrals: %d", response.ReferralLink, response.ReferralCount)
		}
		sections = append(sections, withOptionalPrefix(response.ReferralText, referral))
	}
	return limitTelegramText(strings.Join(sections, "\n\n"))
}

func withOptionalPrefix(prefix, text string) string {
	prefix = boundedCustomText(prefix, 700)
	if prefix == "" {
		return text
	}
	return prefix + "\n" + text
}

func boundedCustomText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes == 1 {
		return "…"
	}
	return strings.TrimSpace(string(runes[:maxRunes-1])) + "…"
}

func limitTelegramText(text string) string {
	runes := []rune(text)
	if len(runes) <= maxTelegramMessageRunes {
		return text
	}
	if maxTelegramMessageRunes <= 1 {
		return string(runes[:maxTelegramMessageRunes])
	}
	return strings.TrimSpace(string(runes[:maxTelegramMessageRunes-1])) + "…"
}

func formatMissingChannels(channels []StartRequiredChannel) string {
	return formatMissingChannelsWithPrefix(channels, "")
}

func formatMissingChannelsWithPrefix(channels []StartRequiredChannel, configuredPrefix string) string {
	prefix := strings.TrimSpace(configuredPrefix)
	if prefix == "" {
		prefix = "Join the required Telegram channels before continuing:"
	}
	suffix := "\n\nAfter joining, tap Recheck below or send /start again."
	truncated := "\n\nAdditional required channels are configured."

	maxPrefixRunes := 1200
	prefix = boundedCustomText(prefix, maxPrefixRunes)

	var builder strings.Builder
	builder.WriteString(prefix)
	used := len([]rune(prefix))
	suffixRunes := len([]rune(suffix))
	truncatedRunes := len([]rune(truncated))
	for index, channel := range channels {
		entry := "\n\n" + channel.DisplayName + "\n" + channel.JoinURL
		if channel.CustomText != "" {
			entry += "\n" + channel.CustomText
		}
		entryRunes := len([]rune(entry))
		if used+entryRunes+suffixRunes > maxTelegramMessageRunes {
			if index < len(channels) && used+truncatedRunes+suffixRunes <= maxTelegramMessageRunes {
				builder.WriteString(truncated)
			}
			break
		}
		builder.WriteString(entry)
		used += entryRunes
	}
	builder.WriteString(suffix)
	return builder.String()
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
