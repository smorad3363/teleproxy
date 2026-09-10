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
	"github.com/smorad3363/teleproxy/internal/referral"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestReferralHistoryAPIRequiresAuthenticationAndIsNoStore(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/referral/history", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized history = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	authorized := performJSON(server.Handler(), http.MethodGet, "/api/referral/history", "", cookie, "")
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized history = %d %s", authorized.Code, authorized.Body.String())
	}
	if got := authorized.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("history Cache-Control = %q", got)
	}
}

func TestReferralHistoryAPIPaginatesWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)
	inviter := resolveHTTPReferralUser(t, ctx, db, 9101)
	invitee1 := resolveHTTPReferralUser(t, ctx, db, 9102)
	invitee2 := resolveHTTPReferralUser(t, ctx, db, 9103)
	base := time.Unix(1_820_000_000, 0).UTC()
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	firstCreated, err := referral.AttributeNewInvitee(ctx, db, invitee1, code.Value, base.Add(time.Second))
	if err != nil || firstCreated.Outcome != referral.OutcomeCreated {
		t.Fatalf("create first attribution = %#v, %v", firstCreated, err)
	}
	secondCreated, err := referral.AttributeNewInvitee(ctx, db, invitee2, code.Value, base.Add(2*time.Second))
	if err != nil || secondCreated.Outcome != referral.OutcomeCreated {
		t.Fatalf("create second attribution = %#v, %v", secondCreated, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, invitee2.User.ID, referral.RejectionCooldown, base.Add(3*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject second attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeSettings := countHTTPRows(t, db, "settings")

	firstResponse := performJSON(server.Handler(), http.MethodGet, "/api/referral/history?limit=1", "", cookie, "")
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first history page = %d %s", firstResponse.Code, firstResponse.Body.String())
	}
	var firstPage struct {
		History      []referral.HistoryEntry `json:"history"`
		NextBeforeID *int64                  `json:"next_before_id"`
	}
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.History) != 1 || firstPage.History[0].ID != secondCreated.Attribution.ID || firstPage.History[0].InviterTelegramID != 9101 || firstPage.History[0].InviteeTelegramID != 9103 {
		t.Fatalf("first history page = %#v", firstPage)
	}
	if firstPage.History[0].Status != referral.StatusRejected || firstPage.History[0].RejectionReason == nil || *firstPage.History[0].RejectionReason != string(referral.RejectionCooldown) {
		t.Fatalf("first history state = %#v", firstPage.History[0])
	}
	if firstPage.NextBeforeID == nil || *firstPage.NextBeforeID != secondCreated.Attribution.ID {
		t.Fatalf("first next_before_id = %v", firstPage.NextBeforeID)
	}

	secondResponse := performJSON(server.Handler(), http.MethodGet, "/api/referral/history?limit=1&before_id="+strconv.FormatInt(*firstPage.NextBeforeID, 10), "", cookie, "")
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second history page = %d %s", secondResponse.Code, secondResponse.Body.String())
	}
	var secondPage struct {
		History      []referral.HistoryEntry `json:"history"`
		NextBeforeID *int64                  `json:"next_before_id"`
	}
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.History) != 1 || secondPage.History[0].ID != firstCreated.Attribution.ID || secondPage.NextBeforeID != nil {
		t.Fatalf("second history page = %#v", secondPage)
	}
	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("history API mutated authoritative state")
	}
}

func TestReferralHistoryAPIRejectsInvalidPagination(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)
	for _, path := range []string{
		"/api/referral/history?before_id=0",
		"/api/referral/history?before_id=-1",
		"/api/referral/history?before_id=nope",
		"/api/referral/history?before_id=1&before_id=2",
		"/api/referral/history?limit=0",
		"/api/referral/history?limit=" + strconv.Itoa(referral.MaxHistoryLimit+1),
		"/api/referral/history?unknown=1",
	} {
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"REFERRAL_HISTORY_INVALID"`) {
			t.Fatalf("invalid history query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func authenticatedReferralHistoryAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie) {
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
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}
}

func resolveHTTPReferralUser(t *testing.T, ctx context.Context, db *sql.DB, telegramID int64) telegramuser.ResolveResult {
	t.Helper()
	resolved, err := telegramuser.Resolve(ctx, db, telegramID, time.Unix(1_819_999_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Created {
		t.Fatalf("telegramuser.Resolve(%d) Created = false", telegramID)
	}
	return resolved
}

func countHTTPRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
