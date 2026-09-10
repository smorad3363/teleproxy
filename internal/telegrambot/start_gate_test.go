package telegrambot

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/smorad3363/teleproxy/internal/forcedjoin"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/settings"
)

type fakeStartMembership struct {
	memberships map[string]bool
	err         error
	calls       []string
}

func (f *fakeStartMembership) IsChatMember(_ context.Context, chatRef string, _ int64) (bool, error) {
	f.calls = append(f.calls, chatRef)
	if f.err != nil {
		return false, f.err
	}
	return f.memberships[chatRef], nil
}

type gatedStartProvisioner struct {
	calls int
}

func (p *gatedStartProvisioner) Ensure(_ context.Context, username string) (proxyprovision.Result, error) {
	p.calls++
	return proxyprovision.Result{
		Username:  username,
		Link:      "tg://proxy?server=proxy.example&port=443&secret=00112233445566778899aabbccddeeff",
		SyncState: proxyuser.SyncPending,
	}, nil
}

func TestForcedJoinBlocksGiftAndProvisionUntilAllRequiredMembershipsPass(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 1, 2, 3, 4, 5, 0, time.UTC)
	createForcedJoinChannel(t, db, "@later", "Later", "https://t.me/later", true, true, 20, now)
	createForcedJoinChannel(t, db, "@first", "First", "https://t.me/first", true, true, 10, now)
	createForcedJoinChannel(t, db, "@optional", "Optional", "https://t.me/optional", true, false, 1, now)
	createForcedJoinChannel(t, db, "@disabled", "Disabled", "https://t.me/disabled", false, true, 2, now)

	membership := &fakeStartMembership{memberships: map[string]bool{"@first": false, "@later": false}}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "TeleProxyBot", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{UpdateID: 1, Message: &Message{From: &TelegramUser{ID: 8101}, Chat: Chat{ID: 8101, Type: "private"}, Text: "/start ref_A"}}

	blocked, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("blocked Handle() = %#v handled=%v err=%v", blocked, handled, err)
	}
	if len(blocked.MissingChannels) != 2 || blocked.MissingChannels[0].DisplayName != "First" || blocked.MissingChannels[1].DisplayName != "Later" {
		t.Fatalf("missing channels = %#v", blocked.MissingChannels)
	}
	if strings.Join(membership.calls, ",") != "@first,@later" {
		t.Fatalf("membership calls = %#v", membership.calls)
	}
	if provisioner.calls != 0 || blocked.InitialGiftBytes != 0 || blocked.ProxyLink != "" {
		t.Fatalf("blocked response provisioned/granted unexpectedly: %#v calls=%d", blocked, provisioner.calls)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM telegram_users WHERE telegram_id = ?", 1, int64(8101))
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 0, "start-gift:telegram:8101")

	membership.memberships["@first"] = true
	membership.memberships["@later"] = true
	membership.calls = nil
	ready, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("ready Handle() = %#v handled=%v err=%v", ready, handled, err)
	}
	if len(ready.MissingChannels) != 0 || ready.InitialGiftBytes != settings.DefaultStartGiftBytes || ready.RemainingBytes != settings.DefaultStartGiftBytes || ready.ProxyLink == "" {
		t.Fatalf("ready response = %#v", ready)
	}
	if provisioner.calls != 1 {
		t.Fatalf("provision calls = %d, want 1", provisioner.calls)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 1, "start-gift:telegram:8101")

	if _, _, err := app.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 1, "start-gift:telegram:8101")
}

func TestForcedJoinMembershipFailureFailsClosedAfterIdentityResolution(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 2, 3, 4, 5, 6, 0, time.UTC)
	createForcedJoinChannel(t, db, "@required", "Required", "https://t.me/required", true, true, 1, now)
	membership := &fakeStartMembership{err: errors.New("Telegram unavailable")}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{Message: &Message{From: &TelegramUser{ID: 8102}, Chat: Chat{ID: 8102, Type: "private"}, Text: "/start"}}
	if _, handled, err := app.Handle(context.Background(), update); err == nil || !handled {
		t.Fatalf("Handle() handled=%v err=%v", handled, err)
	}
	if provisioner.calls != 0 {
		t.Fatalf("provision calls = %d, want 0", provisioner.calls)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM telegram_users WHERE telegram_id = ?", 1, int64(8102))
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 0, "start-gift:telegram:8102")
}

func TestForcedJoinWithNoRequiredChannelsPreservesProvisionedStart(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 3, 4, 5, 6, 7, 0, time.UTC)
	membership := &fakeStartMembership{memberships: map[string]bool{}}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{Message: &Message{From: &TelegramUser{ID: 8103}, Chat: Chat{ID: 8103, Type: "private"}, Text: "/start"}}
	response, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("Handle() = %#v handled=%v err=%v", response, handled, err)
	}
	if response.InitialGiftBytes != settings.DefaultStartGiftBytes || response.ProxyLink == "" || provisioner.calls != 1 || len(membership.calls) != 0 {
		t.Fatalf("response = %#v membership=%#v provision calls=%d", response, membership.calls, provisioner.calls)
	}
}

func TestForcedJoinConstructorAndMissingChannelRendering(t *testing.T) {
	db := startAppTestDB(t)
	if _, err := NewStartApplicationWithForcedJoin(db, "", nil, &gatedStartProvisioner{}, nil); err == nil {
		t.Fatal("NewStartApplicationWithForcedJoin accepted nil membership client")
	}
	response := StartResponse{MissingChannels: []StartRequiredChannel{
		{DisplayName: "First", JoinURL: "https://t.me/first"},
		{DisplayName: "Second", JoinURL: "https://t.me/second"},
	}}
	text := formatStartResponse(response)
	if !strings.Contains(text, "First") || !strings.Contains(text, "https://t.me/first") || !strings.Contains(text, "send /start again") || strings.Contains(text, "Proxy link provisioning") {
		t.Fatalf("missing-channel message = %q", text)
	}

	many := StartResponse{MissingChannels: make([]StartRequiredChannel, 30)}
	for i := range many.MissingChannels {
		many.MissingChannels[i] = StartRequiredChannel{DisplayName: strings.Repeat("x", 128), JoinURL: "https://t.me/" + strings.Repeat("y", 200)}
	}
	if got := utf8.RuneCountInString(formatStartResponse(many)); got > 4096 {
		t.Fatalf("missing-channel message has %d runes, want <= 4096", got)
	}
}

func createForcedJoinChannel(t *testing.T, db *sql.DB, chatRef, displayName, joinURL string, enabled, required bool, position int, now time.Time) {
	t.Helper()
	_, err := forcedjoin.Create(context.Background(), db, forcedjoin.CreateChannel{
		ChatRef:     chatRef,
		DisplayName: displayName,
		JoinURL:     joinURL,
		Enabled:     enabled,
		Required:    required,
		Position:    position,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
}

func assertStartGateCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for %q", got, want, query)
	}
}
