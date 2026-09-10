package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/sponsor"
)

func TestSponsorAdminRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedSponsorAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/sponsors", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingCSRF := performJSON(server.Handler(), http.MethodPost, "/api/sponsors", validSponsorJSON("One", "@one", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true, 1), cookie, "")
	if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("missing CSRF = %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestSponsorAdminCRUDAndOrdering(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedSponsorAPI(t, db)

	first := performJSON(server.Handler(), http.MethodPost, "/api/sponsors", validSponsorJSON("One", "@Sponsor_One", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true, 3), cookie, csrf)
	if first.Code != http.StatusCreated {
		t.Fatalf("create first = %d %s", first.Code, first.Body.String())
	}
	second := performJSON(server.Handler(), http.MethodPost, "/api/sponsors", validSponsorJSON("Two", "https://T.ME/sponsor_two", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", false, 1), cookie, csrf)
	if second.Code != http.StatusCreated {
		t.Fatalf("create second = %d %s", second.Code, second.Body.String())
	}

	var firstBody struct {
		Profile sponsor.Profile `json:"profile"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}
	var secondBody struct {
		Profile sponsor.Profile `json:"profile"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatal(err)
	}
	if firstBody.Profile.ChannelRef != "@sponsor_one" || firstBody.Profile.AdTag != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("canonical first profile = %#v", firstBody.Profile)
	}

	listed := performJSON(server.Handler(), http.MethodGet, "/api/sponsors", "", cookie, "")
	if listed.Code != http.StatusOK || listed.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("list = %d %s headers=%v", listed.Code, listed.Body.String(), listed.Header())
	}
	var listBody struct {
		Profiles []sponsor.Profile `json:"profiles"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Profiles) != 2 || listBody.Profiles[0].ID != firstBody.Profile.ID || listBody.Profiles[1].ID != secondBody.Profile.ID {
		t.Fatalf("list order = %#v", listBody.Profiles)
	}

	updated := performJSON(server.Handler(), http.MethodPut, "/api/sponsors/"+strconv.FormatInt(firstBody.Profile.ID, 10), `{"name":"One updated","channel_ref":"@one_updated","ad_tag":"cccccccccccccccccccccccccccccccc","enabled":false,"weight":7,"starts_at":"2030-01-01T00:00:00Z","ends_at":"2030-01-02T00:00:00Z","notes":"scheduled"}`, cookie, csrf)
	if updated.Code != http.StatusOK {
		t.Fatalf("update = %d %s", updated.Code, updated.Body.String())
	}
	if !strings.Contains(updated.Body.String(), `"name":"One updated"`) || !strings.Contains(updated.Body.String(), `"weight":7`) || !strings.Contains(updated.Body.String(), `"enabled":false`) {
		t.Fatalf("update body = %s", updated.Body.String())
	}

	deleted := performJSON(server.Handler(), http.MethodDelete, "/api/sponsors/"+strconv.FormatInt(secondBody.Profile.ID, 10), "", cookie, csrf)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", deleted.Code, deleted.Body.String())
	}
	listed = performJSON(server.Handler(), http.MethodGet, "/api/sponsors", "", cookie, "")
	if strings.Contains(listed.Body.String(), `"name":"Two"`) || !strings.Contains(listed.Body.String(), `"name":"One updated"`) {
		t.Fatalf("list after delete = %s", listed.Body.String())
	}
}

func TestSponsorAdminRejectsInvalidConflictNotFoundAndOversizedWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedSponsorAPI(t, db)

	created := performJSON(server.Handler(), http.MethodPost, "/api/sponsors", validSponsorJSON("One", "@one", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true, 1), cookie, csrf)
	if created.Code != http.StatusCreated {
		t.Fatalf("create fixture = %d %s", created.Code, created.Body.String())
	}
	before := sponsorCountHTTP(t, db)

	tests := []struct {
		name string
		body string
		code int
		key  string
	}{
		{name: "unknown field", body: `{"name":"Bad","channel_ref":"@bad","ad_tag":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","enabled":true,"weight":1,"unknown":1}`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "missing enabled", body: `{"name":"Bad","channel_ref":"@bad","ad_tag":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","weight":1}`, code: http.StatusBadRequest, key: `"code":"SPONSOR_INVALID"`},
		{name: "bad channel", body: validSponsorJSON("Bad", "https://example.com/bad", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", true, 1), code: http.StatusBadRequest, key: `"code":"SPONSOR_INVALID"`},
		{name: "bad tag", body: validSponsorJSON("Bad", "@bad", "nothex", true, 1), code: http.StatusBadRequest, key: `"code":"SPONSOR_INVALID"`},
		{name: "bad weight", body: validSponsorJSON("Bad", "@bad", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", true, 0), code: http.StatusBadRequest, key: `"code":"SPONSOR_INVALID"`},
		{name: "bad window", body: `{"name":"Bad","channel_ref":"@bad","ad_tag":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","enabled":true,"weight":1,"starts_at":"2030-01-02T00:00:00Z","ends_at":"2030-01-01T00:00:00Z","notes":""}`, code: http.StatusBadRequest, key: `"code":"SPONSOR_INVALID"`},
		{name: "oversized", body: `{"name":"` + strings.Repeat("x", int(maxSponsorProfileBodyBytes)) + `","channel_ref":"@big","ad_tag":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","enabled":true,"weight":1}`, code: http.StatusRequestEntityTooLarge, key: `"code":"REQUEST_TOO_LARGE"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performJSON(server.Handler(), http.MethodPost, "/api/sponsors", test.body, cookie, csrf)
			if response.Code != test.code || !strings.Contains(response.Body.String(), test.key) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if sponsorCountHTTP(t, db) != before {
				t.Fatal("invalid sponsor request mutated state")
			}
		})
	}

	conflict := performJSON(server.Handler(), http.MethodPost, "/api/sponsors", validSponsorJSON("Duplicate", "@duplicate", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true, 2), cookie, csrf)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"SPONSOR_CONFLICT"`) {
		t.Fatalf("conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	if sponsorCountHTTP(t, db) != before {
		t.Fatal("conflict mutated sponsor count")
	}

	notFound := performJSON(server.Handler(), http.MethodDelete, "/api/sponsors/999999", "", cookie, csrf)
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), `"code":"SPONSOR_NOT_FOUND"`) {
		t.Fatalf("not found = %d %s", notFound.Code, notFound.Body.String())
	}
	badID := performJSON(server.Handler(), http.MethodDelete, "/api/sponsors/nope", "", cookie, csrf)
	if badID.Code != http.StatusBadRequest || !strings.Contains(badID.Body.String(), `"code":"SPONSOR_INVALID_ID"`) {
		t.Fatalf("bad id = %d %s", badID.Code, badID.Body.String())
	}
}

func authenticatedSponsorAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie, string) {
	t.Helper()
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}, sessionCSRF(token)
}

func validSponsorJSON(name, channelRef, adTag string, enabled bool, weight int64) string {
	return `{"name":"` + name + `","channel_ref":"` + channelRef + `","ad_tag":"` + adTag + `","enabled":` + strconv.FormatBool(enabled) + `,"weight":` + strconv.FormatInt(weight, 10) + `,"starts_at":null,"ends_at":null,"notes":""}`
}

func sponsorCountHTTP(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sponsor_profiles").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
