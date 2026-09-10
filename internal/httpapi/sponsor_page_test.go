package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/sponsor"
)

func TestSponsorPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/sponsors", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated sponsor page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "csrf-token") {
		t.Fatal("unauthenticated sponsor page exposed CSRF markup")
	}
}

func TestSponsorPageRendersCanonicalProfilesEscapedAndNoStore(t *testing.T) {
	db := testDB(t)
	server, cookies, csrf := authenticatedSponsorAPI(t, db)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	startsAt := now.Add(time.Hour)
	endsAt := startsAt.Add(24 * time.Hour)
	profile, err := sponsor.Create(context.Background(), db, sponsor.CreateProfile{
		Name:       `Alpha <script>alert("x")</script>`,
		ChannelRef: "@Sponsor_Test",
		AdTag:      "ABCDEFABCDEFABCDEFABCDEFABCDEFAB",
		Enabled:    true,
		Weight:     7,
		StartsAt:   &startsAt,
		EndsAt:     &endsAt,
		Notes:      `<img src=x onerror="alert(1)">`,
	}, now)
	if err != nil {
		t.Fatal(err)
	}

	response := perform(server.Handler(), http.MethodGet, "/sponsors", nil, cookies)
	if response.Code != http.StatusOK {
		t.Fatalf("sponsor page = %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Sponsor Profiles",
		"@sponsor_test",
		"abcdefabcdefabcdefabcdefabcdefab",
		"2030-01-01T01:00:00Z",
		"2030-01-02T01:00:00Z",
		`X-CSRF-Token`,
		`/api/sponsors`,
		`content="` + csrf + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("sponsor page missing %q: %s", want, body)
		}
	}
	if !strings.Contains(body, `data-id="`+strconv.FormatInt(profile.ID, 10)+`"`) {
		t.Fatalf("sponsor page missing profile id %d", profile.ID)
	}
	if strings.Contains(body, `<script>alert("x")</script>`) || strings.Contains(body, `<img src=x onerror="alert(1)">`) {
		t.Fatalf("sponsor page rendered unescaped profile content: %s", body)
	}
	if !strings.Contains(body, `Alpha &lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`) {
		t.Fatalf("sponsor name was not safely escaped: %s", body)
	}
	if !strings.Contains(body, `&lt;img src=x onerror=&#34;alert(1)&#34;&gt;`) {
		t.Fatalf("sponsor notes were not safely escaped: %s", body)
	}
}

func TestDashboardLinksToSponsorPage(t *testing.T) {
	db := testDB(t)
	server, cookies, _ := authenticatedSponsorAPI(t, db)

	response := perform(server.Handler(), http.MethodGet, "/", nil, cookies)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `href="/sponsors"`) {
		t.Fatalf("dashboard does not link to Sponsor Profiles: %s", response.Body.String())
	}
}
