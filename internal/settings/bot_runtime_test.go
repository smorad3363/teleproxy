package settings

import (
	"context"
	"testing"
	"time"
)

func TestBotRuntimeDefaultsAndRoundTrip(t *testing.T) {
	db := settingsTestDB(t)
	ctx := context.Background()

	if got, ok, err := BotRuntime(ctx, db); err != nil || ok || got != (BotRuntimeSettings{}) {
		t.Fatalf("BotRuntime default = %#v ok=%v err=%v", got, ok, err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	stored, err := SetBotRuntime(ctx, db, BotRuntimeSettings{
		Username:    "@Teleproxy_TestBot",
		AdminChatID: 123456789,
		Enabled:     true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Username != "Teleproxy_TestBot" || stored.AdminChatID != 123456789 || !stored.Enabled || stored.UpdatedAt != now.Unix() {
		t.Fatalf("stored = %#v", stored)
	}
	got, ok, err := BotRuntime(ctx, db)
	if err != nil || !ok {
		t.Fatalf("BotRuntime = %#v ok=%v err=%v", got, ok, err)
	}
	if got != stored {
		t.Fatalf("round trip = %#v, want %#v", got, stored)
	}
}

func TestBotRuntimeRejectsInvalidValuesWithoutMutation(t *testing.T) {
	db := settingsTestDB(t)
	ctx := context.Background()
	baseline, err := SetBotRuntime(ctx, db, BotRuntimeSettings{Username: "TeleproxyBot", AdminChatID: 42, Enabled: false}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	for _, value := range []BotRuntimeSettings{
		{Username: "", AdminChatID: 42},
		{Username: "bad-name", AdminChatID: 42},
		{Username: "TeleproxyBot", AdminChatID: 0},
		{Username: "TeleproxyBot", AdminChatID: -1},
	} {
		if _, err := SetBotRuntime(ctx, db, value, time.Unix(200, 0)); err == nil {
			t.Fatalf("SetBotRuntime accepted %#v", value)
		}
		got, ok, err := BotRuntime(ctx, db)
		if err != nil || !ok || got != baseline {
			t.Fatalf("invalid update mutated settings: got=%#v ok=%v err=%v", got, ok, err)
		}
	}
}
