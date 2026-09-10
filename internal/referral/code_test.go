package referral

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEnsureCodeIsStableAndConcurrent(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	resolved := resolveTestUser(t, ctx, db, 10001)
	now := time.Unix(1_800_000_000, 0).UTC()

	const workers = 12
	results := make(chan Code, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, err := EnsureCode(ctx, db, resolved.User.ID, now)
			if err != nil {
				errs <- err
				return
			}
			results <- code
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("EnsureCode() concurrent error = %v", err)
	}

	var value string
	for code := range results {
		if !validCode(code.Value) {
			t.Fatalf("generated code %q is invalid", code.Value)
		}
		if value == "" {
			value = code.Value
		}
		if code.Value != value {
			t.Fatalf("concurrent code = %q, want stable %q", code.Value, value)
		}
	}
	if value == "" {
		t.Fatal("no referral code returned")
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM referral_codes WHERE telegram_user_id = ?", resolved.User.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("referral code rows = %d, want 1", count)
	}
	stored, err := GetCode(ctx, db, resolved.User.ID)
	if err != nil {
		t.Fatalf("GetCode() error = %v", err)
	}
	if stored.Value != value {
		t.Fatalf("stored code = %q, want %q", stored.Value, value)
	}
}

func TestEnsureCodeIsUniqueAcrossUsers(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	first := resolveTestUser(t, ctx, db, 11001)
	second := resolveTestUser(t, ctx, db, 11002)
	now := time.Unix(1_800_000_100, 0).UTC()

	firstCode, err := EnsureCode(ctx, db, first.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	secondCode, err := EnsureCode(ctx, db, second.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if firstCode.Value == secondCode.Value {
		t.Fatal("different Telegram users received the same referral code")
	}
}

func TestReferralSchemaEnforcesCodeUniqueness(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	first := resolveTestUser(t, ctx, db, 13501)
	second := resolveTestUser(t, ctx, db, 13502)

	if _, err := db.ExecContext(ctx, `
INSERT INTO referral_codes(telegram_user_id, code, created_at)
VALUES (?, 'ABCDEFGHIJKLMNOPQRSTUVWX', 1)`, first.User.ID); err != nil {
		t.Fatalf("insert first referral code: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO referral_codes(telegram_user_id, code, created_at)
VALUES (?, 'ABCDEFGHIJKLMNOPQRSTUVWX', 1)`, second.User.ID); err == nil {
		t.Fatal("duplicate referral code unexpectedly succeeded")
	}
}
