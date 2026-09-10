package telemt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const testUserSecret = "00112233445566778899aabbccddeeff"

func TestHealthUsesBearerAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"status":"ok","read_only":true},"revision":"abc"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got := client.Health(context.Background())
	if got.State != StateHealthy || !got.ReadOnly {
		t.Fatalf("Health() = %#v", got)
	}
}

func TestHealthClassifiesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got := client.Health(context.Background()); got.State != StateUnauthorized {
		t.Fatalf("Health() state = %q", got.State)
	}
}

func TestHealthRejectsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"data":{"status":"wrong"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got := client.Health(context.Background()); got.State != StateInvalidResponse {
		t.Fatalf("Health() state = %q", got.State)
	}
}

func TestHealthClassifiesUnavailableAndTimeout(t *testing.T) {
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client, err := New(closed.URL, testToken, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	if got := client.Health(context.Background()); got.State != StateUnavailable {
		t.Fatalf("closed server state = %q", got.State)
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"ok":true,"data":{"status":"ok","read_only":false}}`))
	}))
	defer slow.Close()
	client, err = New(slow.URL, testToken, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got := client.Health(context.Background()); got.State != StateUnavailable {
		t.Fatalf("timeout state = %q", got.State)
	}
}

func TestNewFromTokenFileRequiresOwnerOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(testToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromTokenFile("http://telemt:9091", path, time.Second); err != nil {
		t.Fatalf("NewFromTokenFile() error = %v", err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromTokenFile("http://telemt:9091", path, time.Second); err == nil {
		t.Fatal("NewFromTokenFile() accepted group-readable token")
	}
}

func TestConstructorErrorsDoNotContainToken(t *testing.T) {
	_, err := New("bad-url", testToken, time.Second)
	if err == nil {
		t.Fatal("New() error = nil")
	}
	if strings.Contains(err.Error(), testToken) {
		t.Fatal("constructor error leaked token")
	}
}

func TestUserLifecycleRequests(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/users":
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
			var body struct {
				Username string `json:"username"`
				Enabled  bool   `json:"enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Username != "alice" || !body.Enabled {
				t.Fatalf("create body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"username":"alice","enabled":true,"in_runtime":true},"secret":"` + testUserSecret + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users":
			_, _ = w.Write([]byte(`{"ok":true,"data":[{"username":"alice","enabled":true,"in_runtime":true}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/users/alice/disable":
			_, _ = w.Write([]byte(`{"ok":true,"data":{"username":"alice","enabled":false,"in_runtime":true}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/users/alice/rotate-secret":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"username":"alice","enabled":false,"in_runtime":true},"secret":"` + testUserSecret + `"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := client.CreateUser(context.Background(), "alice", true)
	if err != nil || credential.Secret != testUserSecret {
		t.Fatalf("CreateUser() = %#v, %v", credential, err)
	}
	users, err := client.ListUsers(context.Background())
	if err != nil || len(users) != 1 || users[0].Username != "alice" {
		t.Fatalf("ListUsers() = %#v, %v", users, err)
	}
	user, err := client.SetUserEnabled(context.Background(), "alice", false)
	if err != nil || user.Enabled {
		t.Fatalf("SetUserEnabled() = %#v, %v", user, err)
	}
	rotated, err := client.RotateUserSecret(context.Background(), "alice")
	if err != nil || rotated.Secret != testUserSecret {
		t.Fatalf("RotateUserSecret() = %#v, %v", rotated, err)
	}
	if calls.Load() != 4 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestLifecycleFailureIsClassifiedWithoutBodyLeak(t *testing.T) {
	const upstreamSecret = "do-not-leak-this-upstream-body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"user_exists","message":"` + upstreamSecret + `"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateUser(context.Background(), "alice", true)
	if FailureCodeOf(err) != FailureConflict {
		t.Fatalf("failure code = %q, err = %v", FailureCodeOf(err), err)
	}
	if strings.Contains(err.Error(), upstreamSecret) {
		t.Fatal("error leaked upstream response body")
	}
}

func TestLifecycleRejectsInvalidCredentialResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"username":"alice","enabled":true,"in_runtime":true},"secret":"bad"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateUser(context.Background(), "alice", true)
	if FailureCodeOf(err) != FailureInvalidOutput {
		t.Fatalf("failure code = %q, err = %v", FailureCodeOf(err), err)
	}
}

func TestLifecycleValidatesUsernameBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateUser(context.Background(), "bad user", true); err == nil {
		t.Fatal("CreateUser() accepted invalid username")
	}
	if calls.Load() != 0 {
		t.Fatalf("network calls = %d", calls.Load())
	}
}
