package telemt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetUserLinksReturnsValidatedLinksWithoutSeparateSecret(t *testing.T) {
	classic := "tg://proxy?server=203.0.113.10&port=443&secret=00112233445566778899aabbccddeeff"
	tls := "tg://proxy?server=proxy.example.com&port=443&secret=ee00112233445566778899aabbccddeeff6578616d706c652e636f6d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/users/alice" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"username":"alice","enabled":true,"in_runtime":true,"links":{"classic":["` + classic + `"],"secure":[],"tls":["` + tls + `"],"tls_domains":[{"domain":"example.com","link":"` + tls + `"}]},"secret":"must-not-be-decoded"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	links, err := client.GetUserLinks(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	all := links.All()
	if len(all) != 3 || all[0] != classic || all[1] != tls || all[2] != tls {
		t.Fatalf("links = %#v", links)
	}
}

func TestGetUserLinksRejectsInvalidOutput(t *testing.T) {
	for _, response := range []string{
		`{"ok":true,"data":{"username":"other","links":{"classic":[],"secure":[],"tls":[],"tls_domains":[]}}}`,
		`{"ok":true,"data":{"username":"alice","links":{"classic":["https://example.com/secret"],"secure":[],"tls":[],"tls_domains":[]}}}`,
		`{"ok":true,"data":{"username":"alice","links":{"classic":["tg://proxy?server=x&port=0&secret=00112233445566778899aabbccddeeff"],"secure":[],"tls":[],"tls_domains":[]}}}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(response))
		}))
		client, err := New(server.URL, testToken, time.Second)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		_, err = client.GetUserLinks(context.Background(), "alice")
		server.Close()
		if FailureCodeOf(err) != FailureInvalidOutput {
			t.Fatalf("response %s error = %v", response, err)
		}
	}
}

func TestGetUserLinksValidatesUsernameBeforeNetwork(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	client, err := New(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetUserLinks(context.Background(), "bad/name"); err == nil {
		t.Fatal("invalid username accepted")
	}
	if calls != 0 {
		t.Fatalf("invalid username made %d network calls", calls)
	}
}
