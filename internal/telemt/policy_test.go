package telemt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPatchUserPolicyPreservesSetZeroAndSetExpiration(t *testing.T) {
	const expiration = "2030-01-02T03:04:05Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/users/alice" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 2 || body["expiration_rfc3339"] != expiration {
			t.Fatalf("patch body = %#v", body)
		}
		quota, ok := body["data_quota_bytes"].(float64)
		if !ok || quota != 0 {
			t.Fatalf("quota = %#v", body["data_quota_bytes"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"username":"alice","enabled":true,"in_runtime":true,"expiration_rfc3339":"2030-01-02T03:04:05Z","data_quota_bytes":0,"total_octets":12}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	state, err := client.PatchUserPolicy(context.Background(), "alice", UserPolicyPatch{
		Expiration: SetExpirationRFC3339(expiration),
		DataQuota:  SetDataQuotaBytes(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.DataQuotaBytes == nil || *state.DataQuotaBytes != 0 || state.ExpirationRFC3339 == nil || *state.ExpirationRFC3339 != expiration {
		t.Fatalf("state = %#v", state)
	}
}

func TestPatchUserPolicyClearEmitsExplicitNull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if string(body["expiration_rfc3339"]) != "null" || string(body["data_quota_bytes"]) != "null" {
			t.Fatalf("clear body = %#v", body)
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"username":"alice","enabled":true,"in_runtime":true,"expiration_rfc3339":null,"data_quota_bytes":null,"total_octets":0}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.PatchUserPolicy(context.Background(), "alice", UserPolicyPatch{
		Expiration: ClearExpiration(),
		DataQuota:  ClearDataQuota(),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyValidationRejectsBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.PatchUserPolicy(context.Background(), "alice", UserPolicyPatch{Expiration: SetExpirationRFC3339("not-a-date")}); err == nil {
		t.Fatal("invalid expiration accepted")
	}
	if _, err := client.PatchUserPolicy(context.Background(), "alice", UserPolicyPatch{}); err == nil {
		t.Fatal("empty policy patch accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("network calls = %d", calls.Load())
	}
}

func TestGetUserPolicyValidatesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"data":{"username":"alice","enabled":true,"in_runtime":true,"expiration_rfc3339":"bad-date","data_quota_bytes":1024,"total_octets":10}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetUserPolicy(context.Background(), "alice")
	if FailureCodeOf(err) != FailureInvalidOutput {
		t.Fatalf("failure code = %q, err=%v", FailureCodeOf(err), err)
	}
}

func TestResetUserQuotaUsesEmptyBodyAndParsesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/users/alice/reset-quota" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.ContentLength > 0 || r.Header.Get("Content-Type") != "" {
			t.Fatalf("reset request unexpectedly has body/content type: len=%d content-type=%q", r.ContentLength, r.Header.Get("Content-Type"))
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"username":"alice","used_bytes":0,"last_reset_epoch_secs":1893456000}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	reset, err := client.ResetUserQuota(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if reset.Username != "alice" || reset.UsedBytes != 0 || reset.LastResetEpochSecs != 1893456000 {
		t.Fatalf("reset = %#v", reset)
	}
}

func TestPolicyHTTPFailureDoesNotLeakUpstreamBody(t *testing.T) {
	const secret = "upstream-secret-like-message"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"bad_request","message":"` + secret + `"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.PatchUserPolicy(context.Background(), "alice", UserPolicyPatch{DataQuota: SetDataQuotaBytes(1024)})
	if FailureCodeOf(err) != FailureRejected {
		t.Fatalf("failure code = %q, err=%v", FailureCodeOf(err), err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("policy error leaked upstream body")
	}
}
