package telemt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const (
	createWithSecretToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	callerProvidedSecret = "00112233445566778899aabbccddeeff"
)

func TestCreateUserWithSecretSendsExactSecretDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/users" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+createWithSecretToken {
			t.Fatalf("Authorization = %q", got)
		}
		var body struct {
			Username string `json:"username"`
			Secret   string `json:"secret"`
			Enabled  bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Username != "tg_7" || body.Secret != callerProvidedSecret || body.Enabled {
			t.Fatalf("create body = %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"username":"tg_7","enabled":false,"in_runtime":true},"secret":"` + callerProvidedSecret + `"}}`))
	}))
	defer server.Close()

	client, err := New(server.URL, createWithSecretToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := client.CreateUserWithSecret(context.Background(), "tg_7", false, callerProvidedSecret)
	if err != nil {
		t.Fatalf("CreateUserWithSecret() error = %v", err)
	}
	if credential.User.Username != "tg_7" || credential.User.Enabled || credential.Secret != callerProvidedSecret {
		t.Fatalf("credential = %#v", credential)
	}
}

func TestCreateUserWithSecretRejectsInvalidSecretBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()
	client, err := New(server.URL, createWithSecretToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateUserWithSecret(context.Background(), "tg_7", false, "not-a-secret")
	if FailureCodeOf(err) != FailureRejected {
		t.Fatalf("failure = %q, err=%v", FailureCodeOf(err), err)
	}
	if calls.Load() != 0 {
		t.Fatalf("network calls = %d, want 0", calls.Load())
	}
}

func TestCreateUserWithSecretRejectsDifferentReturnedSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"username":"tg_7","enabled":false,"in_runtime":true},"secret":"ffeeddccbbaa99887766554433221100"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, createWithSecretToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateUserWithSecret(context.Background(), "tg_7", false, callerProvidedSecret)
	if FailureCodeOf(err) != FailureInvalidOutput {
		t.Fatalf("failure = %q, err=%v", FailureCodeOf(err), err)
	}
}
