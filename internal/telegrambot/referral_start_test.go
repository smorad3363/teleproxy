package telegrambot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/referral"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestGatedStartPersistsReferralBeforeRecheckAndSurfacesStableSummary(t *testing.T) {
	ctx := context.Background()
	db := startAppTestDB(t)
	now := time.Date(2034, 1, 2, 3, 4, 5, 0, time.UTC)

	inviter, err := telegramuser.Resolve(ctx, db, 9101, now)
	if err != nil || !inviter.Created {
		t.Fatalf("resolve inviter = %#v, %v", inviter, err)
	}
	inviterCode, err := referral.EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatalf("ensure inviter code: %v", err)
	}
	createForcedJoinChannel(t, db, "@required", "Required", "https://t.me/required", true, true, 1, now)

	membership := &fakeStartMembership{memberships: map[string]bool{"@required": false}}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "TeleProxyBot", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	start := Update{UpdateID: 1, Message: &Message{
		MessageID: 10,
		From:      &TelegramUser{ID: 9102, FirstName: "Invitee"},
		Chat:      Chat{ID: 9102, Type: "private"},
		Text:      "/start " + inviterCode.Value,
	}}

	blocked, handled, err := app.Handle(ctx, start)
	if err != nil || !handled || len(blocked.MissingChannels) != 1 {
		t.Fatalf("blocked Handle() = %#v handled=%v err=%v", blocked, handled, err)
	}
	if blocked.ReferralCode != "" || blocked.ReferralLink != "" || blocked.InitialGiftBytes != 0 || provisioner.calls != 0 {
		t.Fatalf("blocked response crossed entitlement/summary boundary: %#v calls=%d", blocked, provisioner.calls)
	}
	invitee, err := telegramuser.Get(ctx, db, 9102)
	if err != nil {
		t.Fatalf("read invitee: %v", err)
	}
	attribution, err := referral.GetAttribution(ctx, db, invitee.ID)
	if err != nil {
		t.Fatalf("read pending attribution: %v", err)
	}
	if attribution.InviterUserID != inviter.User.ID || attribution.InviteeUserID != invitee.ID || attribution.Status != referral.StatusPending {
		t.Fatalf("pending attribution = %#v", attribution)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", 0, invitee.ProxyUserID)

	membership.memberships["@required"] = true
	recheck := Update{UpdateID: 2, CallbackQuery: &CallbackQuery{
		ID:   "callback-9102",
		From: &TelegramUser{ID: 9102, FirstName: "Invitee"},
		Message: &Message{
			MessageID: 11,
			Chat:      Chat{ID: 9102, Type: "private"},
		},
		Data: forcedJoinRecheckCallbackData,
	}}
	ready, handled, err := app.Handle(ctx, recheck)
	if err != nil || !handled {
		t.Fatalf("recheck Handle() = %#v handled=%v err=%v", ready, handled, err)
	}
	if ready.ReferralCode == "" || ready.ReferralLink != "https://t.me/TeleProxyBot?start="+ready.ReferralCode || ready.ReferralCount != 0 {
		t.Fatalf("referral summary = code=%q link=%q count=%d", ready.ReferralCode, ready.ReferralLink, ready.ReferralCount)
	}
	if !strings.Contains(formatStartResponse(ready), ready.ReferralLink) || !strings.Contains(formatStartResponse(ready), "Successful referrals: 0") {
		t.Fatalf("formatted start response missing referral summary: %q", formatStartResponse(ready))
	}
	if provisioner.calls != 1 {
		t.Fatalf("provision calls = %d, want 1", provisioner.calls)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", 1, invitee.ProxyUserID)

	after, err := referral.GetAttribution(ctx, db, invitee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.ID != attribution.ID || after.InviterUserID != attribution.InviterUserID || after.Status != referral.StatusPending {
		t.Fatalf("recheck changed pending attribution: before=%#v after=%#v", attribution, after)
	}
}

func TestGatedStartUnknownAndLateReferralPayloadsDoNotAttribute(t *testing.T) {
	ctx := context.Background()
	db := startAppTestDB(t)
	now := time.Date(2034, 2, 3, 4, 5, 6, 0, time.UTC)

	inviter, err := telegramuser.Resolve(ctx, db, 9201, now)
	if err != nil {
		t.Fatal(err)
	}
	inviterCode, err := referral.EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	membership := &fakeStartMembership{memberships: map[string]bool{}}
	app, err := NewStartApplicationWithForcedJoin(db, "TeleProxyBot", func() time.Time { return now }, &gatedStartProvisioner{}, membership)
	if err != nil {
		t.Fatal(err)
	}

	unknownCode := "AAAAAAAAAAAAAAAAAAAAAAAA"
	if unknownCode == inviterCode.Value {
		unknownCode = "BBBBBBBBBBBBBBBBBBBBBBBB"
	}
	unknownStart := Update{UpdateID: 3, Message: &Message{From: &TelegramUser{ID: 9202}, Chat: Chat{ID: 9202, Type: "private"}, Text: "/start " + unknownCode}}
	unknownResponse, handled, err := app.Handle(ctx, unknownStart)
	if err != nil || !handled || unknownResponse.ReferralCode == "" {
		t.Fatalf("unknown referral start = %#v handled=%v err=%v", unknownResponse, handled, err)
	}
	unknownUser, err := telegramuser.Get(ctx, db, 9202)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := referral.GetAttribution(ctx, db, unknownUser.ID); !errors.Is(err, referral.ErrAttributionNotFound) {
		t.Fatalf("unknown payload attribution error = %v, want not found", err)
	}

	plainStart := Update{UpdateID: 4, Message: &Message{From: &TelegramUser{ID: 9203}, Chat: Chat{ID: 9203, Type: "private"}, Text: "/start"}}
	if _, handled, err := app.Handle(ctx, plainStart); err != nil || !handled {
		t.Fatalf("plain start handled=%v err=%v", handled, err)
	}
	lateStart := plainStart
	lateStart.UpdateID = 5
	lateStart.Message.Text = "/start " + inviterCode.Value
	if _, handled, err := app.Handle(ctx, lateStart); err != nil || !handled {
		t.Fatalf("late referral start handled=%v err=%v", handled, err)
	}
	lateUser, err := telegramuser.Get(ctx, db, 9203)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := referral.GetAttribution(ctx, db, lateUser.ID); !errors.Is(err, referral.ErrAttributionNotFound) {
		t.Fatalf("late payload attribution error = %v, want not found", err)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", 1, lateUser.ProxyUserID)
}
