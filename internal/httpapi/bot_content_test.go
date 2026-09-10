package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/smorad3363/teleproxy/internal/botcontent"
)

func TestBotContentAPIRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/bot-content", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized GET = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingPutCSRF := performJSON(server.Handler(), http.MethodPut, "/api/bot-content/welcome", `{"text":"hello"}`, cookie, "")
	if missingPutCSRF.Code != http.StatusForbidden || !strings.Contains(missingPutCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("PUT without CSRF = %d %s", missingPutCSRF.Code, missingPutCSRF.Body.String())
	}
	missingDeleteCSRF := performJSON(server.Handler(), http.MethodDelete, "/api/bot-content/welcome", "", cookie, "")
	if missingDeleteCSRF.Code != http.StatusForbidden || !strings.Contains(missingDeleteCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("DELETE without CSRF = %d %s", missingDeleteCSRF.Code, missingDeleteCSRF.Body.String())
	}
}

func TestBotContentAPICRUDAndDeterministicList(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)

	empty := performJSON(server.Handler(), http.MethodGet, "/api/bot-content", "", cookie, "")
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), `"content":[]`) {
		t.Fatalf("empty GET = %d %s", empty.Code, empty.Body.String())
	}
	if got := empty.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("empty GET Cache-Control = %q", got)
	}

	welcome := performJSON(server.Handler(), http.MethodPut, "/api/bot-content/welcome", `{"text":"سلام 👋"}`, cookie, csrf)
	if welcome.Code != http.StatusOK || !strings.Contains(welcome.Body.String(), `"slot":"welcome"`) || !strings.Contains(welcome.Body.String(), `"text":"سلام 👋"`) {
		t.Fatalf("welcome PUT = %d %s", welcome.Code, welcome.Body.String())
	}
	support := performJSON(server.Handler(), http.MethodPut, "/api/bot-content/support", `{"text":"Support"}`, cookie, csrf)
	if support.Code != http.StatusOK {
		t.Fatalf("support PUT = %d %s", support.Code, support.Body.String())
	}

	listed := performJSON(server.Handler(), http.MethodGet, "/api/bot-content", "", cookie, "")
	if listed.Code != http.StatusOK {
		t.Fatalf("GET = %d %s", listed.Code, listed.Body.String())
	}
	body := listed.Body.String()
	supportIndex := strings.Index(body, `"slot":"support"`)
	welcomeIndex := strings.Index(body, `"slot":"welcome"`)
	if supportIndex < 0 || welcomeIndex < 0 || supportIndex >= welcomeIndex {
		t.Fatalf("Bot Content list is not deterministic: %s", body)
	}

	deleted := performJSON(server.Handler(), http.MethodDelete, "/api/bot-content/support", "", cookie, csrf)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("DELETE = %d %s", deleted.Code, deleted.Body.String())
	}
	if got := deleted.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("DELETE Cache-Control = %q", got)
	}
	if _, err := botcontent.Get(ctx, db, botcontent.SlotSupport); !errors.Is(err, botcontent.ErrNotFound) {
		t.Fatalf("cleared support Get() error = %v", err)
	}
	stored, err := botcontent.Get(ctx, db, botcontent.SlotWelcome)
	if err != nil || stored.Text != "سلام 👋" {
		t.Fatalf("welcome after support DELETE = %#v, %v", stored, err)
	}
}

func TestBotContentAPIRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeReferrals := countHTTPRows(t, db, "referral_attributions")

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		code   string
	}{
		{name: "malformed", method: http.MethodPut, path: "/api/bot-content/welcome", body: `{"text":`, status: http.StatusBadRequest, code: `"code":"BAD_REQUEST"`},
		{name: "unknown field", method: http.MethodPut, path: "/api/bot-content/welcome", body: `{"text":"ok","unknown":1}`, status: http.StatusBadRequest, code: `"code":"BAD_REQUEST"`},
		{name: "trailing JSON", method: http.MethodPut, path: "/api/bot-content/welcome", body: `{"text":"ok"}{}`, status: http.StatusBadRequest, code: `"code":"BAD_REQUEST"`},
		{name: "empty text", method: http.MethodPut, path: "/api/bot-content/welcome", body: `{"text":""}`, status: http.StatusBadRequest, code: `"code":"BOT_CONTENT_INVALID"`},
		{name: "too many runes", method: http.MethodPut, path: "/api/bot-content/welcome", body: `{"text":"` + strings.Repeat("🙂", botcontent.MaxTextRunes+1) + `"}`, status: http.StatusBadRequest, code: `"code":"BOT_CONTENT_INVALID"`},
		{name: "unknown slot put", method: http.MethodPut, path: "/api/bot-content/other", body: `{"text":"ok"}`, status: http.StatusBadRequest, code: `"code":"BOT_CONTENT_INVALID"`},
		{name: "unknown slot delete", method: http.MethodDelete, path: "/api/bot-content/other", status: http.StatusBadRequest, code: `"code":"BOT_CONTENT_INVALID"`},
		{name: "missing override", method: http.MethodDelete, path: "/api/bot-content/welcome", status: http.StatusNotFound, code: `"code":"BOT_CONTENT_NOT_FOUND"`},
		{name: "oversized", method: http.MethodPut, path: "/api/bot-content/welcome", body: strings.Repeat(" ", int(maxBotContentBodyBytes)+1) + `{}`, status: http.StatusRequestEntityTooLarge, code: `"code":"REQUEST_TOO_LARGE"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performJSON(server.Handler(), test.method, test.path, test.body, cookie, csrf)
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.code) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
	if got := countHTTPRows(t, db, "bot_content"); got != 0 {
		t.Fatalf("invalid requests mutated bot_content: count=%d", got)
	}
	if countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "referral_attributions") != beforeReferrals {
		t.Fatal("Bot Content API mutated unrelated authoritative state")
	}
}
