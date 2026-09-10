package settings

import (
	"context"
	"database/sql"
	"testing"
)

func TestReferralRewardDefaultsAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := settingsTestDB(t)

	got, err := ReferralReward(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bytes != DefaultReferralRewardBytes || got.ExpiryDays != DefaultReferralRewardDays {
		t.Fatalf("default referral reward = %#v", got)
	}

	if err := SetReferralReward(ctx, db, 3_500_000_000, 21); err != nil {
		t.Fatal(err)
	}
	got, err = ReferralReward(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bytes != 3_500_000_000 || got.ExpiryDays != 21 {
		t.Fatalf("configured referral reward = %#v", got)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	fromTx, err := ReferralRewardTx(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	if fromTx != got {
		t.Fatalf("transactional referral reward = %#v, want %#v", fromTx, got)
	}
}

func TestReferralRewardRejectsInvalidInputsAndCorruptStorage(t *testing.T) {
	ctx := context.Background()
	db := settingsTestDB(t)

	for _, test := range []struct {
		bytes int64
		days  int64
	}{
		{bytes: 0, days: 14},
		{bytes: -1, days: 14},
		{bytes: 2_000_000_000, days: 0},
		{bytes: 2_000_000_000, days: -1},
	} {
		if err := SetReferralReward(ctx, db, test.bytes, test.days); err == nil {
			t.Fatalf("SetReferralReward(%d, %d) accepted invalid input", test.bytes, test.days)
		}
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO settings(key, value, updated_at)
VALUES (?, 'invalid', 0)`, referralRewardBytesKey); err != nil {
		t.Fatal(err)
	}
	if _, err := ReferralReward(ctx, db); err == nil {
		t.Fatal("ReferralReward accepted corrupt byte setting")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM settings WHERE key = ?", referralRewardBytesKey); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO settings(key, value, updated_at)
VALUES (?, '0', 0)`, referralRewardExpiryDaysKey); err != nil {
		t.Fatal(err)
	}
	if _, err := ReferralReward(ctx, db); err == nil {
		t.Fatal("ReferralReward accepted corrupt expiry setting")
	}
}

func TestSetReferralRewardIsAtomic(t *testing.T) {
	ctx := context.Background()
	db := settingsTestDB(t)

	if _, err := db.ExecContext(ctx, `
CREATE TRIGGER reject_referral_reward_expiry
BEFORE INSERT ON settings
WHEN NEW.key = 'referral_reward_expiry_days'
BEGIN
    SELECT RAISE(ABORT, 'blocked by test');
END`); err != nil {
		t.Fatal(err)
	}
	if err := SetReferralReward(ctx, db, 4_000_000_000, 30); err == nil {
		t.Fatal("SetReferralReward unexpectedly succeeded")
	}

	var count int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM settings
WHERE key IN (?, ?)`, referralRewardBytesKey, referralRewardExpiryDaysKey).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed atomic update persisted %d referral settings", count)
	}
	got, err := ReferralReward(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bytes != DefaultReferralRewardBytes || got.ExpiryDays != DefaultReferralRewardDays {
		t.Fatalf("failed atomic update changed defaults: %#v", got)
	}
}

func TestReferralRewardSettingsDoNotMutateReferralOrCreditState(t *testing.T) {
	ctx := context.Background()
	db := settingsTestDB(t)

	if err := SetReferralReward(ctx, db, 2_500_000_000, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := ReferralReward(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"referral_attributions", "credit_buckets"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("settings mutated %s: count=%d", table, count)
		}
	}
}
