package telegramuser

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestStartIsIdempotentUnderConcurrentReplay(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2031, 1, 2, 3, 4, 5, 0, time.UTC)
	const telegramID int64 = 123456789
	const workers = 12

	results := make(chan StartResult, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := Start(context.Background(), db, telegramID, now)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Start() error = %v", err)
	}

	created := 0
	var first StartResult
	for result := range results {
		if first.User.ID == 0 {
			first = result
		}
		if result.Created {
			created++
		}
		if result.User.TelegramID != telegramID || result.User.ProxyUsername != "tg_123456789" {
			t.Fatalf("unexpected result = %#v", result)
		}
		if result.InitialGiftBytes != settings.DefaultStartGiftBytes {
			t.Fatalf("gift bytes = %d", result.InitialGiftBytes)
		}
	}
	if created != 1 {
		t.Fatalf("created results = %d, want 1", created)
	}

	assertCount(t, db, "SELECT COUNT(*) FROM telegram_users WHERE telegram_id = ?", 1, telegramID)
	assertCount(t, db, "SELECT COUNT(*) FROM proxy_users WHERE username = ?", 1, "tg_123456789")
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ? AND idempotency_key = ?", 1, first.User.ProxyUserID, startGiftKey(telegramID))
}

func TestStartGiftSettingAffectsOnlyNewUsers(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2031, 2, 1, 0, 0, 0, 0, time.UTC)

	first, err := Start(context.Background(), db, 101, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.InitialGiftBytes != settings.DefaultStartGiftBytes {
		t.Fatalf("first gift = %d", first.InitialGiftBytes)
	}
	const changed = int64(200_000_000)
	if err := settings.SetStartGiftBytes(context.Background(), db, changed); err != nil {
		t.Fatal(err)
	}
	second, err := Start(context.Background(), db, 202, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.InitialGiftBytes != changed {
		t.Fatalf("second gift = %d, want %d", second.InitialGiftBytes, changed)
	}
	replayed, err := Start(context.Background(), db, 101, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created || replayed.InitialGiftBytes != settings.DefaultStartGiftBytes {
		t.Fatalf("replayed first user = %#v", replayed)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ? AND idempotency_key = ?", 1, first.User.ProxyUserID, startGiftKey(101))
}

func TestStartRejectsInvalidIdentitySettingAndProxyNameCollision(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2031, 3, 1, 0, 0, 0, 0, time.UTC)
	for _, telegramID := range []int64{0, -1} {
		if _, err := Start(context.Background(), db, telegramID, now); err == nil {
			t.Fatalf("Start accepted Telegram ID %d", telegramID)
		}
	}

	if _, err := db.Exec(`INSERT INTO settings(key, value, updated_at) VALUES ('start_gift_bytes', '0', 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(context.Background(), db, 303, now); err == nil {
		t.Fatal("Start accepted invalid stored gift setting")
	}
	assertCount(t, db, "SELECT COUNT(*) FROM telegram_users", 0)
	if _, err := db.Exec("DELETE FROM settings WHERE key = 'start_gift_bytes'"); err != nil {
		t.Fatal(err)
	}

	if _, err := proxyuser.Create(context.Background(), db, "tg_404", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(context.Background(), db, 404, now); !errors.Is(err, ErrProxyUsernameConflict) {
		t.Fatalf("proxy username collision error = %v", err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM telegram_users WHERE telegram_id = 404", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE idempotency_key = ?", 0, startGiftKey(404))
}

func TestGetRejectsMissingAndReturnsStoredIdentity(t *testing.T) {
	db := telegramTestDB(t)
	if _, err := Get(context.Background(), db, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing error = %v", err)
	}
	now := time.Date(2031, 4, 1, 0, 0, 0, 0, time.UTC)
	started, err := Start(context.Background(), db, 999, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Get(context.Background(), db, 999)
	if err != nil {
		t.Fatal(err)
	}
	if got != started.User {
		t.Fatalf("Get() = %#v, want %#v", got, started.User)
	}
}

func telegramTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func assertCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for %q", got, want, query)
	}
}
