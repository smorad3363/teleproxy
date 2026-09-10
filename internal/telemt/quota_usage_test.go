package telemt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetUserQuotaUsageFindsTargetAndUsesBearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stats/users/quota" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"users":[{"username":"alice","data_quota_bytes":4096,"used_bytes":1024,"last_reset_epoch_secs":123},{"username":"bob","data_quota_bytes":8192,"used_bytes":2048,"last_reset_epoch_secs":456}]}}`))
	}))
	defer server.Close()

	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	usage, found, err := client.GetUserQuotaUsage(context.Background(), "bob")
	if err != nil {
		t.Fatal(err)
	}
	if !found || usage.Username != "bob" || usage.DataQuotaBytes != 8192 || usage.UsedBytes != 2048 || usage.LastResetEpochSecs != 456 {
		t.Fatalf("GetUserQuotaUsage() = %#v found=%v", usage, found)
	}
}

func TestGetUserQuotaUsageReturnsCleanAbsence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"data":{"users":[]}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	usage, found, err := client.GetUserQuotaUsage(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if found || usage != (QuotaUsage{}) {
		t.Fatalf("GetUserQuotaUsage() = %#v found=%v", usage, found)
	}
}

func TestListQuotaUsageRejectsMalformedEntries(t *testing.T) {
	for name, payload := range map[string]string{
		"zero quota":   `{"ok":true,"data":{"users":[{"username":"alice","data_quota_bytes":0,"used_bytes":0,"last_reset_epoch_secs":1}]}}`,
		"invalid user": `{"ok":true,"data":{"users":[{"username":"bad user","data_quota_bytes":1,"used_bytes":0,"last_reset_epoch_secs":1}]}}`,
		"duplicate":    `{"ok":true,"data":{"users":[{"username":"alice","data_quota_bytes":1,"used_bytes":0,"last_reset_epoch_secs":1},{"username":"alice","data_quota_bytes":2,"used_bytes":0,"last_reset_epoch_secs":1}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(payload))
			}))
			defer server.Close()
			client, err := New(server.URL, testToken, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.ListQuotaUsage(context.Background()); FailureCodeOf(err) != FailureInvalidOutput {
				t.Fatalf("ListQuotaUsage() error = %v", err)
			}
		})
	}
}

func TestListQuotaUsageDoesNotSurfaceUpstreamBody(t *testing.T) {
	const marker = "sensitive-upstream-detail"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(marker))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListQuotaUsage(context.Background())
	if FailureCodeOf(err) != FailureUnavailable {
		t.Fatalf("ListQuotaUsage() error = %v", err)
	}
	if strings.Contains(err.Error(), marker) {
		t.Fatal("upstream response body leaked through error")
	}
}
