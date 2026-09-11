package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunHealthcheckAcceptsHTTP200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}))
	defer server.Close()

	if err := runHealthcheck(server.URL); err != nil {
		t.Fatalf("runHealthcheck() error = %v", err)
	}
}

func TestRunHealthcheckRejectsNon200WithoutResponseBodyLeak(t *testing.T) {
	const secretBody = "sensitive-health-detail"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(secretBody))
	}))
	defer server.Close()

	err := runHealthcheck(server.URL)
	if err == nil {
		t.Fatal("runHealthcheck() error = nil, want non-200 failure")
	}
	if !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("runHealthcheck() error = %q, want bounded status", err)
	}
	if strings.Contains(err.Error(), secretBody) || strings.Contains(err.Error(), server.URL) {
		t.Fatalf("runHealthcheck() leaked response body or URL: %q", err)
	}
}

func TestRunHealthcheckDoesNotFollowRedirects(t *testing.T) {
	var targetHit atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetHit.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()

	err := runHealthcheck(redirect.URL)
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("runHealthcheck() error = %v, want redirect failure", err)
	}
	if targetHit.Load() {
		t.Fatal("runHealthcheck() followed redirect")
	}
}

func TestRunHealthcheckTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := runHealthcheckWithTimeout(server.URL, 20*time.Millisecond)
	if err == nil || err.Error() != "control healthcheck request failed" {
		t.Fatalf("runHealthcheckWithTimeout() error = %v, want bounded timeout failure", err)
	}
	if strings.Contains(err.Error(), server.URL) {
		t.Fatalf("runHealthcheckWithTimeout() leaked URL: %q", err)
	}
}

func TestRunHealthcheckRejectsInvalidURL(t *testing.T) {
	for _, rawURL := range []string{
		"",
		"/readyz",
		"https://127.0.0.1/readyz",
		"http://user:password@127.0.0.1/readyz",
	} {
		if err := runHealthcheck(rawURL); err == nil {
			t.Fatalf("runHealthcheck(%q) error = nil", rawURL)
		}
	}
}

func TestRunControlCommandDispatch(t *testing.T) {
	handled, err := runControlCommand(nil)
	if handled || err != nil {
		t.Fatalf("runControlCommand(nil) = %v, %v", handled, err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	handled, err = runControlCommand([]string{"healthcheck", server.URL})
	if !handled || err != nil {
		t.Fatalf("healthcheck command = %v, %v", handled, err)
	}

	for _, args := range [][]string{
		{"healthcheck"},
		{"healthcheck", server.URL, "extra"},
		{"unknown", server.URL},
	} {
		handled, err = runControlCommand(args)
		if !handled || err == nil {
			t.Fatalf("runControlCommand(%q) = %v, %v, want handled failure", args, handled, err)
		}
	}
}
