package telegrambot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeStartHandler struct {
	response StartResponse
	handled  bool
	err      error
	calls    int
	last     Update
}

func (f *fakeStartHandler) Handle(_ context.Context, update Update) (StartResponse, bool, error) {
	f.calls++
	f.last = update
	return f.response, f.handled, f.err
}

type fakeMessageSender struct {
	calls  int
	chatID int64
	text   string
	err    error
}

func (f *fakeMessageSender) SendMessage(_ context.Context, chatID int64, text string) (SentMessage, error) {
	f.calls++
	f.chatID = chatID
	f.text = text
	if f.err != nil {
		return SentMessage{}, f.err
	}
	return SentMessage{MessageID: int64(f.calls), Chat: Chat{ID: chatID}, Text: text}, nil
}

func TestLoadWebhookSecretFileRequiresOwnerOnlyValidSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webhook-secret")
	if err := os.WriteFile(path, []byte("Webhook_secret-123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadWebhookSecretFile(path)
	if err != nil {
		t.Fatalf("LoadWebhookSecretFile() error = %v", err)
	}
	if got != "Webhook_secret-123" {
		t.Fatalf("secret = %q", got)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWebhookSecretFile(path); err == nil {
		t.Fatal("LoadWebhookSecretFile() accepted group-readable file")
	}
}

func TestLoadWebhookSecretFileRejectsInvalidContentWithoutLeak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webhook-secret")
	secret := "invalid secret value"
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadWebhookSecretFile(path)
	if err == nil {
		t.Fatal("LoadWebhookSecretFile() error = nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("error leaked webhook secret")
	}
}

func TestWebhookRejectsWrongSecretBeforeApplication(t *testing.T) {
	start := &fakeStartHandler{handled: true}
	sender := &fakeMessageSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if start.calls != 0 || sender.calls != 0 {
		t.Fatalf("unexpected calls start=%d sender=%d", start.calls, sender.calls)
	}
}

func TestWebhookProcessesValidStartAndSendsSafeMessage(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{
		ChatID: 42, TelegramID: 7, ProxyUsername: "tg_7", Created: true, RemainingBytes: 100000000,
	}}
	sender := &fakeMessageSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	req := webhookRequest(`{"update_id":11,"message":{"message_id":5,"from":{"id":7,"is_bot":false,"first_name":"A"},"chat":{"id":42,"type":"private"},"date":1,"text":"/start"}}`, "secret_123")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if start.calls != 1 || start.last.UpdateID != 11 {
		t.Fatalf("start calls=%d update=%#v", start.calls, start.last)
	}
	if sender.calls != 1 || sender.chatID != 42 {
		t.Fatalf("sender = %#v", sender)
	}
	if !strings.Contains(sender.text, "tg_7") || !strings.Contains(sender.text, "100000000") {
		t.Fatalf("message = %q", sender.text)
	}
	if strings.Contains(sender.text, "secret_123") {
		t.Fatal("response leaked webhook secret")
	}
}

func TestWebhookDoesNotRetryAfterOutboundFailure(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{ChatID: 42, ProxyUsername: "tg_7", RemainingBytes: 1}}
	sender := &fakeMessageSender{err: &APIError{Code: FailureUnavailable}}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, webhookRequest(`{"update_id":12,"message":{"message_id":5,"from":{"id":7,"is_bot":false,"first_name":"A"},"chat":{"id":42,"type":"private"},"date":1,"text":"/start"}}`, "secret_123"))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if start.calls != 1 || sender.calls != 1 {
		t.Fatalf("calls start=%d sender=%d", start.calls, sender.calls)
	}
}

func TestWebhookApplicationFailureRequestsRetry(t *testing.T) {
	start := &fakeStartHandler{handled: true, err: errors.New("db unavailable")}
	sender := &fakeMessageSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, webhookRequest(`{"update_id":13,"message":{"message_id":5,"from":{"id":7,"is_bot":false,"first_name":"A"},"chat":{"id":42,"type":"private"},"date":1,"text":"/start"}}`, "secret_123"))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls = %d", sender.calls)
	}
}

func TestWebhookBoundsMethodContentTypeJSONAndBody(t *testing.T) {
	start := &fakeStartHandler{}
	sender := &fakeMessageSender{}
	handler, err := newWebhookHandler("secret_123", start, sender, webhookOptions{MaxBodyBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, method, contentType, body string
		want                            int
	}{
		{name: "method", method: http.MethodGet, contentType: "application/json", body: `{}`, want: http.StatusMethodNotAllowed},
		{name: "content type", method: http.MethodPost, contentType: "text/plain", body: `{}`, want: http.StatusUnsupportedMediaType},
		{name: "malformed", method: http.MethodPost, contentType: "application/json", body: `{`, want: http.StatusBadRequest},
		{name: "oversized", method: http.MethodPost, contentType: "application/json", body: strings.Repeat("x", 64), want: http.StatusRequestEntityTooLarge},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/telegram/webhook", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret_123")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}
	if start.calls != 0 || sender.calls != 0 {
		t.Fatalf("unexpected calls start=%d sender=%d", start.calls, sender.calls)
	}
}

func TestWebhookRateLimitsAuthenticatedSources(t *testing.T) {
	now := time.Unix(100, 0)
	start := &fakeStartHandler{handled: false}
	sender := &fakeMessageSender{}
	handler, err := newWebhookHandler("secret_123", start, sender, webhookOptions{
		PerSourceSecond: 1, GlobalSecond: 2, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	first := webhookRequest(`{"update_id":21}`, "secret_123")
	first.RemoteAddr = "192.0.2.1:1000"
	second := webhookRequest(`{"update_id":22}`, "secret_123")
	second.RemoteAddr = "192.0.2.1:1001"
	third := webhookRequest(`{"update_id":23}`, "secret_123")
	third.RemoteAddr = "192.0.2.2:1002"
	fourth := webhookRequest(`{"update_id":24}`, "secret_123")
	fourth.RemoteAddr = "192.0.2.3:1003"
	for index, tc := range []struct {
		req  *http.Request
		want int
	}{{first, http.StatusNoContent}, {second, http.StatusTooManyRequests}, {third, http.StatusNoContent}, {fourth, http.StatusTooManyRequests}} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, tc.req)
		if recorder.Code != tc.want {
			t.Fatalf("request %d status = %d, want %d", index, recorder.Code, tc.want)
		}
	}
	now = now.Add(time.Second)
	reset := webhookRequest(`{"update_id":25}`, "secret_123")
	reset.RemoteAddr = "192.0.2.1:1004"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, reset)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status after window reset = %d", recorder.Code)
	}
}

func webhookRequest(body, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	return req
}
