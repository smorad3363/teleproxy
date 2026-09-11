package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrateVersionTenPreservesReferralAttribution(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "v9.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at INTEGER NOT NULL
)`); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"001_core.sql",
		"002_proxy_users.sql",
		"003_credit_buckets.sql",
		"004_quota_reconciliation.sql",
		"005_quota_accounting.sql",
		"006_telegram_start.sql",
		"007_proxy_provisioning.sql",
		"008_forced_join.sql",
		"009_referrals.sql",
	} {
		version, err := migrationVersion(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(ctx, db, version, name); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}

	inviterProxy := insertMigrationProxyUser(t, db, "ref_inviter", 100)
	inviteeProxy := insertMigrationProxyUser(t, db, "ref_invitee", 100)
	inviterUser := insertMigrationTelegramUser(t, db, 88001, inviterProxy, 200)
	inviteeUser := insertMigrationTelegramUser(t, db, 88002, inviteeProxy, 200)
	if _, err := db.ExecContext(ctx, `
INSERT INTO referral_attributions(
    inviter_user_id, invitee_user_id, status, rejection_reason, finalized_at, created_at, updated_at
) VALUES (?, ?, 'pending', NULL, NULL, 300, 300)`, inviterUser, inviteeUser); err != nil {
		t.Fatalf("insert v9 referral attribution: %v", err)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("upgrade from v9: %v", err)
	}
	var status string
	var eligibleAt sql.NullInt64
	if err := db.QueryRowContext(ctx, `
SELECT status, eligible_at
FROM referral_attributions
WHERE invitee_user_id = ?`, inviteeUser).Scan(&status, &eligibleAt); err != nil {
		t.Fatalf("read upgraded referral attribution: %v", err)
	}
	if status != "pending" || eligibleAt.Valid {
		t.Fatalf("upgraded referral attribution changed: status=%q eligible_at=%#v", status, eligibleAt)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE referral_attributions
SET eligible_at = 400, updated_at = 400
WHERE invitee_user_id = ?`, inviteeUser); err != nil {
		t.Fatalf("eligible_at column is unusable: %v", err)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("rerun after v10: %v", err)
	}
	var migrationCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 14 {
		t.Fatalf("migration count = %d, want 14", migrationCount)
	}
	if err := db.QueryRowContext(ctx, "SELECT eligible_at FROM referral_attributions WHERE invitee_user_id = ?", inviteeUser).Scan(&eligibleAt); err != nil {
		t.Fatal(err)
	}
	if !eligibleAt.Valid || eligibleAt.Int64 != 400 {
		t.Fatalf("rerun changed eligible_at = %#v, want 400", eligibleAt)
	}
}

func insertMigrationProxyUser(t *testing.T, db *sql.DB, username string, now int64) int64 {
	t.Helper()
	result, err := db.Exec(`
INSERT INTO proxy_users(username, desired_enabled, sync_state, created_at, updated_at)
VALUES (?, 1, 'synced', ?, ?)`, username, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func insertMigrationTelegramUser(t *testing.T, db *sql.DB, telegramID, proxyUserID, now int64) int64 {
	t.Helper()
	result, err := db.Exec(`
INSERT INTO telegram_users(telegram_id, proxy_user_id, created_at, updated_at)
VALUES (?, ?, ?, ?)`, telegramID, proxyUserID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
