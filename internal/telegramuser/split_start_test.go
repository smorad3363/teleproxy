package telegramuser

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestResolveCreatesIdentityWithoutGiftAndEnsureStartGiftIsIdempotent(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2032, 1, 2, 3, 4, 5, 0, time.UTC)
	const telegramID int64 = 7001

	resolved, err := Resolve(context.Background(), db, telegramID, now)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Created || resolved.User.ProxyUsername != "tg_7001" {
		t.Fatalf("Resolve() = %#v", resolved)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM telegram_users WHERE telegram_id = ?", 1, telegramID)
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", 0, resolved.User.ProxyUserID)

	replayed, err := Resolve(context.Background(), db, telegramID, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created || replayed.User != resolved.User {
		t.Fatalf("Resolve replay = %#v, want existing %#v", replayed, resolved.User)
	}

	gift, err := EnsureStartGift(context.Background(), db, telegramID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !gift.Granted || gift.InitialGiftBytes != settings.DefaultStartGiftBytes || gift.User != resolved.User {
		t.Fatalf("EnsureStartGift() = %#v", gift)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ? AND idempotency_key = ?", 1, resolved.User.ProxyUserID, startGiftKey(telegramID))

	const changed = int64(333_000_000)
	if err := settings.SetStartGiftBytes(context.Background(), db, changed); err != nil {
		t.Fatal(err)
	}
	giftReplay, err := EnsureStartGift(context.Background(), db, telegramID, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if giftReplay.Granted || giftReplay.InitialGiftBytes != settings.DefaultStartGiftBytes {
		t.Fatalf("EnsureStartGift replay = %#v", giftReplay)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ? AND idempotency_key = ?", 1, resolved.User.ProxyUserID, startGiftKey(telegramID))
}

func TestEnsureStartGiftIsExactlyOnceUnderConcurrentReplay(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2032, 2, 3, 4, 5, 6, 0, time.UTC)
	const telegramID int64 = 7002
	resolved, err := Resolve(context.Background(), db, telegramID, now)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 10
	results := make(chan StartGiftResult, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gift, err := EnsureStartGift(context.Background(), db, telegramID, now.Add(time.Minute))
			if err != nil {
				errs <- err
				return
			}
			results <- gift
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("EnsureStartGift concurrent error = %v", err)
	}
	granted := 0
	for result := range results {
		if result.Granted {
			granted++
		}
		if result.InitialGiftBytes != settings.DefaultStartGiftBytes {
			t.Fatalf("gift bytes = %d", result.InitialGiftBytes)
		}
	}
	if granted != 1 {
		t.Fatalf("granted results = %d, want 1", granted)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ? AND idempotency_key = ?", 1, resolved.User.ProxyUserID, startGiftKey(telegramID))
}

func TestStartAtomicallyFinishesPreviouslyResolvedIdentity(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2032, 3, 4, 5, 6, 7, 0, time.UTC)
	const telegramID int64 = 7003
	resolved, err := Resolve(context.Background(), db, telegramID, now)
	if err != nil {
		t.Fatal(err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", 0, resolved.User.ProxyUserID)

	started, err := Start(context.Background(), db, telegramID, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if started.Created || started.User != resolved.User || started.InitialGiftBytes != settings.DefaultStartGiftBytes {
		t.Fatalf("Start after Resolve = %#v", started)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ? AND idempotency_key = ?", 1, resolved.User.ProxyUserID, startGiftKey(telegramID))
}

func TestResolveDoesNotDependOnGiftSettingAndEnsureRequiresIdentity(t *testing.T) {
	db := telegramTestDB(t)
	now := time.Date(2032, 4, 5, 6, 7, 8, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO settings(key, value, updated_at) VALUES ('start_gift_bytes', '0', 0)`); err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(context.Background(), db, 7004, now)
	if err != nil {
		t.Fatalf("Resolve with invalid gift setting: %v", err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", 0, resolved.User.ProxyUserID)
	if _, err := EnsureStartGift(context.Background(), db, 7004, now); err == nil {
		t.Fatal("EnsureStartGift accepted invalid stored gift setting")
	}
	if _, err := EnsureStartGift(context.Background(), db, 7999, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("EnsureStartGift missing identity error = %v", err)
	}
}
