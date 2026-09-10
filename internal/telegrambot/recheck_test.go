package telegrambot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestForcedJoinRecheckGrantsExistingGiftExactlyOnce(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 4, 5, 6, 7, 8, 0, time.UTC)
	createForcedJoinChannel(t, db, "@required", "Required", "https://t.me/required", true, true, 1, now)
	membership := &fakeStartMembership{memberships: map[string]bool{"@required": false}}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	start := Update{UpdateID: 1, Message: &Message{From: &TelegramUser{ID: 8201}, Chat: Chat{ID: 8201, Type: "private"}, Text: "/start"}}
	blocked, handled, err := app.Handle(context.Background(), start)
	if err != nil || !handled || len(blocked.MissingChannels) != 1 {
		t.Fatalf("blocked start = %#v handled=%v err=%v", blocked, handled, err)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 0, "start-gift:telegram:8201")

	membership.memberships["@required"] = true
	callback := forcedJoinRecheckUpdate(2, "callback-1", 8201)
	ready, handled, err := app.Handle(context.Background(), callback)
	if err != nil || !handled {
		t.Fatalf("recheck = %#v handled=%v err=%v", ready, handled, err)
	}
	if ready.InitialGiftBytes != settings.DefaultStartGiftBytes || ready.RemainingBytes != settings.DefaultStartGiftBytes || ready.ProxyLink == "" {
		t.Fatalf("ready response = %#v", ready)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 1, "start-gift:telegram:8201")

	if _, handled, err := app.Handle(context.Background(), callback); err != nil || !handled {
		t.Fatalf("replayed recheck handled=%v err=%v", handled, err)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 1, "start-gift:telegram:8201")
}

func TestForcedJoinRecheckRejectsUnsafeContextsWithoutMutation(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 5, 6, 7, 8, 9, 0, time.UTC)
	createForcedJoinChannel(t, db, "@required", "Required", "https://t.me/required", true, true, 1, now)
	membership := &fakeStartMembership{memberships: map[string]bool{"@required": false}}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	start := Update{Message: &Message{From: &TelegramUser{ID: 8202}, Chat: Chat{ID: 8202, Type: "private"}, Text: "/start"}}
	if _, handled, err := app.Handle(context.Background(), start); err != nil || !handled {
		t.Fatalf("initial start handled=%v err=%v", handled, err)
	}
	membership.memberships["@required"] = true
	membership.calls = nil

	valid := forcedJoinRecheckUpdate(10, "callback-safe", 8202)
	cases := []Update{
		{UpdateID: 11, CallbackQuery: &CallbackQuery{ID: "callback", From: &TelegramUser{ID: 8202}, Message: valid.CallbackQuery.Message, Data: "forged"}},
		{UpdateID: 12, CallbackQuery: &CallbackQuery{ID: "callback", From: &TelegramUser{ID: 8202}, Data: forcedJoinRecheckCallbackData}},
		{UpdateID: 13, CallbackQuery: &CallbackQuery{ID: "callback", From: &TelegramUser{ID: 8202}, Message: &Message{MessageID: 1, Chat: Chat{ID: -100, Type: "group"}}, Data: forcedJoinRecheckCallbackData}},
		{UpdateID: 14, CallbackQuery: &CallbackQuery{ID: "callback", From: &TelegramUser{ID: 8202}, Message: &Message{MessageID: 1, Chat: Chat{ID: 9999, Type: "private"}}, Data: forcedJoinRecheckCallbackData}},
		{UpdateID: 15, CallbackQuery: &CallbackQuery{ID: "callback", From: &TelegramUser{ID: 8202, IsBot: true}, Message: valid.CallbackQuery.Message, Data: forcedJoinRecheckCallbackData}},
		{UpdateID: 16, CallbackQuery: &CallbackQuery{ID: "callback", From: &TelegramUser{ID: 8202}, Message: &Message{Chat: Chat{ID: 8202, Type: "private"}}, Data: forcedJoinRecheckCallbackData}},
		forcedJoinRecheckUpdate(17, "callback-unknown", 9999),
	}
	for index, update := range cases {
		if _, handled, err := app.Handle(context.Background(), update); err != nil || handled {
			t.Fatalf("unsafe case %d handled=%v err=%v", index, handled, err)
		}
	}
	if len(membership.calls) != 0 || provisioner.calls != 0 {
		t.Fatalf("unsafe callbacks reached gate/provisioner: memberships=%#v provision=%d", membership.calls, provisioner.calls)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 0, "start-gift:telegram:8202")
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM telegram_users", 1)
}

func TestForcedJoinRecheckMembershipFailureRemainsFailClosed(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 6, 7, 8, 9, 10, 0, time.UTC)
	createForcedJoinChannel(t, db, "@required", "Required", "https://t.me/required", true, true, 1, now)
	membership := &fakeStartMembership{memberships: map[string]bool{"@required": false}}
	provisioner := &gatedStartProvisioner{}
	app, err := NewStartApplicationWithForcedJoin(db, "", func() time.Time { return now }, provisioner, membership)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.Handle(context.Background(), Update{Message: &Message{From: &TelegramUser{ID: 8203}, Chat: Chat{ID: 8203, Type: "private"}, Text: "/start"}}); err != nil {
		t.Fatal(err)
	}
	membership.err = errors.New("membership unavailable")
	if _, handled, err := app.Handle(context.Background(), forcedJoinRecheckUpdate(20, "callback-err", 8203)); err == nil || !handled {
		t.Fatalf("recheck handled=%v err=%v", handled, err)
	}
	assertStartGateCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 0, "start-gift:telegram:8203")
	if provisioner.calls != 0 {
		t.Fatalf("provision calls = %d", provisioner.calls)
	}
}

func forcedJoinRecheckUpdate(updateID int64, callbackID string, telegramID int64) Update {
	return Update{UpdateID: updateID, CallbackQuery: &CallbackQuery{
		ID:   callbackID,
		From: &TelegramUser{ID: telegramID},
		Message: &Message{
			MessageID: 99,
			Chat:      Chat{ID: telegramID, Type: "private"},
		},
		Data: forcedJoinRecheckCallbackData,
	}}
}
