package telegrambot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInteractiveBotMethodsSendKeyboardAndAnswerCallback(t *testing.T) {
	const token = "123456:ABC_def-123"
	var sendPayload map[string]any
	var answerPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		switch r.URL.Path {
		case "/bot" + token + "/sendMessage":
			if err := json.NewDecoder(r.Body).Decode(&sendPayload); err != nil {
				t.Errorf("decode send payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":9,"date":1,"chat":{"id":42,"type":"private"},"text":"join"}}`))
		case "/bot" + token + "/answerCallbackQuery":
			if err := json.NewDecoder(r.Body).Decode(&answerPayload); err != nil {
				t.Errorf("decode answer payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newClient(server.URL, token, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	markup := forcedJoinKeyboard([]StartRequiredChannel{
		{DisplayName: "First", JoinURL: "https://t.me/first"},
		{DisplayName: "Second", JoinURL: "https://telegram.me/second"},
	})
	message, err := client.SendMessageWithInlineKeyboard(context.Background(), 42, "join", markup)
	if err != nil || message.MessageID != 9 {
		t.Fatalf("SendMessageWithInlineKeyboard() = %#v, %v", message, err)
	}
	if got := sendPayload["chat_id"]; got != float64(42) {
		t.Fatalf("chat_id = %#v", got)
	}
	reply, ok := sendPayload["reply_markup"].(map[string]any)
	if !ok {
		t.Fatalf("reply_markup = %#v", sendPayload["reply_markup"])
	}
	rows, ok := reply["inline_keyboard"].([]any)
	if !ok || len(rows) != 3 {
		t.Fatalf("inline keyboard = %#v", reply["inline_keyboard"])
	}
	lastRow := rows[2].([]any)
	lastButton := lastRow[0].(map[string]any)
	if lastButton["callback_data"] != forcedJoinRecheckCallbackData {
		t.Fatalf("callback data = %#v", lastButton["callback_data"])
	}
	if len(forcedJoinRecheckCallbackData) < 1 || len(forcedJoinRecheckCallbackData) > 64 {
		t.Fatalf("recheck callback data length = %d", len(forcedJoinRecheckCallbackData))
	}

	if err := client.AnswerCallbackQuery(context.Background(), "callback-123", "Membership verified."); err != nil {
		t.Fatal(err)
	}
	if answerPayload["callback_query_id"] != "callback-123" || answerPayload["text"] != "Membership verified." {
		t.Fatalf("answer payload = %#v", answerPayload)
	}
}

func TestInteractiveKeyboardRejectsUnsafeJoinURLBeforeNetwork(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := newClient(server.URL, "123456:ABC_def-123", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SendMessageWithInlineKeyboard(context.Background(), 42, "join", InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{
		{Text: "Unsafe", URL: "https://example.com/not-telegram"},
	}}})
	if err == nil {
		t.Fatal("unsafe URL accepted")
	}
	if calls != 0 {
		t.Fatalf("network calls = %d, want 0", calls)
	}
}

func TestInteractiveBotFailuresDoNotLeakTokenBearingURL(t *testing.T) {
	const token = "123456:ABC_def-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream body includes "+token, http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := newClient(server.URL, token, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = client.AnswerCallbackQuery(context.Background(), "callback-123", "")
	if FailureCodeOf(err) != FailureUnavailable {
		t.Fatalf("failure code = %q, err=%v", FailureCodeOf(err), err)
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), server.URL) {
		t.Fatalf("error leaked request URL/token: %v", err)
	}
}
