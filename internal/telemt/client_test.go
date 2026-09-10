package telemt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

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
