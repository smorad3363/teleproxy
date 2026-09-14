package telegrambot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetUpdatesUsesBoundedLongPollingContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/bot"+testBotToken+"/getUpdates" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Offset         int64    `json:"offset"`
			Limit          int      `json:"limit"`
			Timeout        int      `json:"timeout"`
			AllowedUpdates []string `json:"allowed_updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Offset != 12 || body.Limit != 100 || body.Timeout != 25 || len(body.AllowedUpdates) != 2 {
			t.Fatalf("body = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":12},{"update_id":13}]}`))
	}))
	defer server.Close()

	client, err := newClient(server.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	updates, err := client.GetUpdates(context.Background(), 12, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 || updates[0].UpdateID != 12 || updates[1].UpdateID != 13 {
		t.Fatalf("updates = %#v", updates)
	}
}

func TestPollerDeletesWebhookWithoutDroppingPendingUpdatesAndDispatchesStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var deleteCalls atomic.Int64
	var updateCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bot" + testBotToken + "/deleteWebhook":
			deleteCalls.Add(1)
			var body struct {
				DropPendingUpdates bool `json:"drop_pending_updates"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.DropPendingUpdates {
				t.Fatal("poller requested dropping pending Telegram updates")
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		case "/bot" + testBotToken + "/getUpdates":
			call := updateCalls.Add(1)
			if call == 1 {
				_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":21,"message":{"message_id":5,"from":{"id":7,"is_bot":false,"first_name":"A"},"chat":{"id":7,"type":"private"},"date":1,"text":"/start"}}]}`))
				return
			}
			cancel()
			_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := newClient(server.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	start := &fakeStartHandler{handled: true, response: StartResponse{ChatID: 7, ProxyUsername: "tg_7", RemainingBytes: 100}}
	sender := &fakeMessageSender{}
	poller, err := newPoller(client, start, sender, 0, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := poller.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if deleteCalls.Load() != 1 {
		t.Fatalf("deleteWebhook calls = %d", deleteCalls.Load())
	}
	if start.calls != 1 || start.last.UpdateID != 21 {
		t.Fatalf("start calls=%d update=%#v", start.calls, start.last)
	}
	if sender.calls != 1 || sender.chatID != 7 {
		t.Fatalf("sender = %#v", sender)
	}
}

func TestGetUpdatesRejectsOutOfOrderResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":9},{"update_id":8}]}`))
	}))
	defer server.Close()
	client, err := newClient(server.URL, testBotToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetUpdates(context.Background(), 0, 0); FailureCodeOf(err) != FailureInvalidOutput {
		t.Fatalf("error = %v", err)
	}
}
