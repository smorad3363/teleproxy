package telegrambot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type pollingKeyboardSender struct {
	fakeMessageSender
	keyboardCalls int
	keyboardErr   error
}

func (f *pollingKeyboardSender) SendMessageWithInlineKeyboard(_ context.Context, chatID int64, text string, _ InlineKeyboardMarkup) (SentMessage, error) {
	f.keyboardCalls++
	if f.keyboardErr != nil {
		return SentMessage{}, f.keyboardErr
	}
	return SentMessage{MessageID: int64(f.keyboardCalls), Chat: Chat{ID: chatID}, Text: text}, nil
}

func TestPollerDispatchFallsBackToPlainTextWhenKeyboardSendFails(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{ChatID: 42, ProxyUsername: "tg_7", RemainingBytes: 100, ReferralLink: "https://t.me/TeleProxyBot?start=abc"}}
	sender := &pollingKeyboardSender{keyboardErr: &APIError{Code: FailureRejected}}
	poller, err := newPoller(&Client{}, start, sender, 0, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{UpdateID: 1, Message: &Message{From: &TelegramUser{ID: 7}, Chat: Chat{ID: 42, Type: "private"}, Text: "/start"}}
	if err := poller.dispatch(context.Background(), update); err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}
	if sender.keyboardCalls != 1 || sender.calls != 1 || sender.chatID != 42 {
		t.Fatalf("sender keyboard=%d plain=%d chat=%d", sender.keyboardCalls, sender.calls, sender.chatID)
	}
}

func TestPollerDispatchReturnsPlainSendFailure(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{ChatID: 42, ProxyUsername: "tg_7", RemainingBytes: 100}}
	sendErr := &APIError{Code: FailureUnavailable}
	sender := &fakeMessageSender{err: sendErr}
	poller, err := newPoller(&Client{}, start, sender, 0, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{UpdateID: 2, Message: &Message{From: &TelegramUser{ID: 7}, Chat: Chat{ID: 42, Type: "private"}, Text: "/start"}}
	if err := poller.dispatch(context.Background(), update); !errors.Is(err, sendErr) {
		t.Fatalf("dispatch() error = %v, want %v", err, sendErr)
	}
}

func TestPollerStopsOnRejectedGetUpdatesAfterWebhookCleanup(t *testing.T) {
	var deleteCalls atomic.Int64
	var updateCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bot" + testBotToken + "/deleteWebhook":
			deleteCalls.Add(1)
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		case "/bot" + testBotToken + "/getUpdates":
			updateCalls.Add(1)
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":409,"description":"Conflict"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newClient(server.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	poller, err := newPoller(client, &fakeStartHandler{}, &fakeMessageSender{}, 0, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	err = poller.Run(context.Background())
	if FailureCodeOf(errors.Unwrap(err)) != FailureRejected {
		t.Fatalf("Run() error = %v", err)
	}
	if deleteCalls.Load() != 1 || updateCalls.Load() != 1 {
		t.Fatalf("calls delete=%d updates=%d", deleteCalls.Load(), updateCalls.Load())
	}
}
