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

func TestIsChatMemberClassifiesTelegramStatuses(t *testing.T) {
	tests := []struct {
		name       string
		resultJSON string
		want       bool
		wantCode   FailureCode
	}{
		{name: "creator", resultJSON: `{"status":"creator","user":{"id":7}}`, want: true},
		{name: "administrator", resultJSON: `{"status":"administrator","user":{"id":7}}`, want: true},
		{name: "member", resultJSON: `{"status":"member","user":{"id":7}}`, want: true},
		{name: "restricted member", resultJSON: `{"status":"restricted","is_member":true,"user":{"id":7}}`, want: true},
		{name: "restricted not member", resultJSON: `{"status":"restricted","is_member":false,"user":{"id":7}}`, want: false},
		{name: "left", resultJSON: `{"status":"left","user":{"id":7}}`, want: false},
		{name: "kicked", resultJSON: `{"status":"kicked","user":{"id":7}}`, want: false},
		{name: "restricted missing flag", resultJSON: `{"status":"restricted","user":{"id":7}}`, wantCode: FailureInvalidOutput},
		{name: "unknown", resultJSON: `{"status":"future","user":{"id":7}}`, wantCode: FailureInvalidOutput},
		{name: "wrong user", resultJSON: `{"status":"member","user":{"id":8}}`, wantCode: FailureInvalidOutput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/getChatMember") {
					t.Fatalf("path = %q", r.URL.Path)
				}
				var request struct {
					ChatID any   `json:"chat_id"`
					UserID int64 `json:"user_id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				if request.ChatID != "@required" || request.UserID != 7 {
					t.Fatalf("request = %#v", request)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true,"result":` + tc.resultJSON + `}`))
			}))
			defer server.Close()
			client, err := newClient(server.URL, "123:secret", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.IsChatMember(context.Background(), "@required", 7)
			if tc.wantCode != "" {
				if FailureCodeOf(err) != tc.wantCode {
					t.Fatalf("error = %v, code=%q", err, FailureCodeOf(err))
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("IsChatMember() = %v, %v", got, err)
			}
		})
	}
}

func TestIsChatMemberSendsNumericChatIDAsIntegerAndSanitizesFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["chat_id"] != float64(-100123) {
			t.Fatalf("chat_id = %#v", request["chat_id"])
		}
		http.Error(w, `{"ok":false,"error_code":403,"description":"secret details"}`, http.StatusForbidden)
	}))
	defer server.Close()
	client, err := newClient(server.URL, "123:secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.IsChatMember(context.Background(), "-100123", 7)
	if FailureCodeOf(err) != FailureUnauthorized {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "secret details") || strings.Contains(err.Error(), "123:secret") {
		t.Fatalf("error leaked sensitive data: %v", err)
	}
}

func TestIsChatMemberRejectsMalformedAndRateLimitedResponses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantCode FailureCode
	}{
		{name: "malformed", status: http.StatusOK, body: `{`, wantCode: FailureInvalidOutput},
		{name: "missing result", status: http.StatusOK, body: `{"ok":true}`, wantCode: FailureInvalidOutput},
		{name: "rate limited envelope", status: http.StatusOK, body: `{"ok":false,"error_code":429}`, wantCode: FailureRateLimited},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client, err := newClient(server.URL, "123:secret", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.IsChatMember(context.Background(), "@required", 7); FailureCodeOf(err) != tc.wantCode {
				t.Fatalf("error = %v, code=%q", err, FailureCodeOf(err))
			}
		})
	}
}

func TestIsChatMemberRejectsInvalidInputsBeforeNetwork(t *testing.T) {
	client, err := newClient("https://api.telegram.org", "123:secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		chat string
		user int64
	}{{"bad", 7}, {"@ok", 0}, {" 123", 7}, {"+123", 7}, {"00123", 7}, {"0", 7}, {"@bad-name", 7}} {
		if _, err := client.IsChatMember(context.Background(), tc.chat, tc.user); FailureCodeOf(err) != FailureRejected {
			t.Fatalf("input %#v returned %v", tc, err)
		}
	}
}
