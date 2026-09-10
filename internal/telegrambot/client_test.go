package telegrambot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testBotToken = "123456:ABCDEFGHIJKLMNOPQRSTUVWXYZ_1234567890"

func TestTokenFileRequiresPrivateOwnerReadableRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-token")
	if err := os.WriteFile(path, []byte(testBotToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	client, err := NewFromTokenFile(path, time.Second)
	if err != nil || client == nil {
		t.Fatalf("NewFromTokenFile() = %#v, %v", client, err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromTokenFile(path, time.Second); err == nil {
		t.Fatal("group-readable token file was accepted")
	}
	if err := os.Chmod(path, 0o200); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromTokenFile(path, time.Second); err == nil {
		t.Fatal("owner-unreadable token file was accepted")
	}
}

func TestConstructorErrorsNeverContainToken(t *testing.T) {
	_, err := newClient("bad-url", testBotToken, time.Second)
	if err == nil {
		t.Fatal("newClient() error = nil")
	}
	if strings.Contains(err.Error(), testBotToken) {
		t.Fatal("constructor leaked token")
	}
	bad := testBotToken + "/secret"
	_, err = New(bad, time.Second)
	if err == nil {
		t.Fatal("New() accepted unsafe token")
	}
	if strings.Contains(err.Error(), bad) {
		t.Fatal("token validation error leaked token")
	}
}

func TestSendMessageUsesJSONAndParsesResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/bot"+testBotToken+"/sendMessage" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		var body struct {
			ChatID int64  `json:"chat_id"`
			Text   string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.ChatID != 42 || body.Text != "hello" {
			t.Fatalf("body = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":7,"date":1924992000,"chat":{"id":42,"type":"private"},"text":"hello"}}`))
	}))
	defer server.Close()
	client, err := newClient(server.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	message, err := client.SendMessage(context.Background(), 42, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if message.MessageID != 7 || message.Chat.ID != 42 || message.Text != "hello" {
		t.Fatalf("message = %#v", message)
	}
}

func TestSendMessageFailuresAreNarrowAndDoNotLeakResponseOrToken(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		want   FailureCode
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, body: `token=` + testBotToken, want: FailureUnauthorized},
		{name: "rate limited", status: http.StatusTooManyRequests, body: `{"ok":false,"description":"` + testBotToken + `"}`, want: FailureRateLimited},
		{name: "server", status: http.StatusBadGateway, body: `<html>` + testBotToken + `</html>`, want: FailureUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := newClient(server.URL, testBotToken, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SendMessage(context.Background(), 1, "hello")
			if FailureCodeOf(err) != test.want {
				t.Fatalf("failure code = %q, want %q; err=%v", FailureCodeOf(err), test.want, err)
			}
			if strings.Contains(err.Error(), testBotToken) || strings.Contains(err.Error(), test.body) {
				t.Fatalf("error leaked sensitive upstream data: %v", err)
			}
		})
	}
}

func TestSendMessageRejectsMalformedOversizedAndInvalidInput(t *testing.T) {
	malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":`))
	}))
	defer malformed.Close()
	client, err := newClient(malformed.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.SendMessage(context.Background(), 1, "hello"); FailureCodeOf(err) != FailureInvalidOutput {
		t.Fatalf("malformed response error = %v", err)
	}

	oversized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBodyBytes+1)))
	}))
	defer oversized.Close()
	client, err = newClient(oversized.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.SendMessage(context.Background(), 1, "hello"); FailureCodeOf(err) != FailureInvalidOutput {
		t.Fatalf("oversized response error = %v", err)
	}

	if _, err := client.SendMessage(context.Background(), 0, "hello"); err == nil {
		t.Fatal("zero chat ID accepted")
	}
	if _, err := client.SendMessage(context.Background(), 1, ""); err == nil {
		t.Fatal("empty text accepted")
	}
	if _, err := client.SendMessage(context.Background(), 1, strings.Repeat("a", 4097)); err == nil {
		t.Fatal("oversized text accepted")
	}
}

func TestNetworkErrorDoesNotLeakTokenBearingURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client, err := newClient(server.URL, testBotToken, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	_, err = client.SendMessage(context.Background(), 1, "hello")
	if FailureCodeOf(err) != FailureUnavailable {
		t.Fatalf("network error = %v", err)
	}
	if strings.Contains(err.Error(), testBotToken) {
		t.Fatal("network error leaked token-bearing URL")
	}
}
