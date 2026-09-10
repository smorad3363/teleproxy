package telegrambot

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestStartApplicationCreatesAndReplaysSafely(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2032, 1, 2, 3, 4, 5, 0, time.UTC)
	app, err := NewStartApplication(db, "TeleProxyBot", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	update := Update{UpdateID: 1, Message: &Message{
		MessageID: 10,
		From:      &TelegramUser{ID: 123456, FirstName: "Alice"},
		Chat:      Chat{ID: 123456, Type: "private"},
		Text:      "/start ref_ABC-1",
	}}
	first, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("first Handle() = %#v, %v, %v", first, handled, err)
	}
	if !first.Created || first.ProxyUsername != "tg_123456" || first.InitialGiftBytes != settings.DefaultStartGiftBytes || first.RemainingBytes != settings.DefaultStartGiftBytes || first.Payload != "ref_ABC-1" {
		t.Fatalf("first response = %#v", first)
	}
	second, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled || second.Created {
		t.Fatalf("second Handle() = %#v, %v, %v", second, handled, err)
	}
	if second.RemainingBytes != first.RemainingBytes || second.ProxyUsername != first.ProxyUsername {
		t.Fatalf("replay changed response state: first=%#v second=%#v", first, second)
	}
	buckets, err := credit.List(context.Background(), db, first.ProxyUsername, now)
	if err != nil || len(buckets) != 1 {
		t.Fatalf("buckets = %#v, %v", buckets, err)
	}
}

func TestStartApplicationShowsCurrentRemainingBalance(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2032, 2, 1, 0, 0, 0, 0, time.UTC)
	app, err := NewStartApplication(db, "", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	update := Update{Message: &Message{From: &TelegramUser{ID: 77}, Chat: Chat{ID: 77, Type: "private"}, Text: "/start"}}
	first, _, err := app.Handle(context.Background(), update)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := credit.Consume(context.Background(), db, first.ProxyUsername, 25_000_000, now); err != nil {
		t.Fatal(err)
	}
	replayed, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("replay = %#v, %v, %v", replayed, handled, err)
	}
	if replayed.RemainingBytes != 75_000_000 {
		t.Fatalf("remaining = %d, want 75000000", replayed.RemainingBytes)
	}
}

func TestStartApplicationIgnoresUnsafeOrUnrelatedUpdates(t *testing.T) {
	db := startAppTestDB(t)
	app, err := NewStartApplication(db, "TeleProxyBot", nil)
	if err != nil {
		t.Fatal(err)
	}
	updates := []Update{
		{},
		{Message: &Message{From: &TelegramUser{ID: 1}, Chat: Chat{ID: 1, Type: "private"}, Text: "/help"}},
		{Message: &Message{From: &TelegramUser{ID: 1, IsBot: true}, Chat: Chat{ID: 1, Type: "private"}, Text: "/start"}},
		{Message: &Message{From: &TelegramUser{ID: 1}, Chat: Chat{ID: -10, Type: "group"}, Text: "/start@TeleProxyBot"}},
		{Message: &Message{From: &TelegramUser{ID: 1}, Chat: Chat{ID: 1, Type: "private"}, Text: "/start@OtherBot"}},
	}
	for index, update := range updates {
		_, handled, err := app.Handle(context.Background(), update)
		if err != nil || handled {
			t.Fatalf("update %d unexpectedly handled: handled=%v err=%v", index, handled, err)
		}
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM telegram_users").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unsafe/unrelated updates created %d Telegram users", count)
	}
}

func TestStartApplicationPropagatesInvalidStartPayloadWithoutMutation(t *testing.T) {
	db := startAppTestDB(t)
	app, err := NewStartApplication(db, "TeleProxyBot", nil)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{Message: &Message{From: &TelegramUser{ID: 1}, Chat: Chat{ID: 1, Type: "private"}, Text: "/start invalid!"}}
	_, handled, err := app.Handle(context.Background(), update)
	if !handled || !errors.Is(err, ErrInvalidStartPayload) {
		t.Fatalf("Handle() handled=%v err=%v", handled, err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM telegram_users").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid payload created %d users", count)
	}
}

func startAppTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
