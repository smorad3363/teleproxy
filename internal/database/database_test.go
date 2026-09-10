package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAppliesSQLiteInvariantsAndMigrations(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "teleproxy.db")
	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertPragmaInt(t, db, "foreign_keys", 1)
	assertPragmaInt(t, db, "synchronous", 1)
	assertPragmaInt(t, db, "busy_timeout", busyTimeoutMS)

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	info, err := os.Stat(filepath.Join(filepath.Dir(dbPath), "teleproxy.db"))
	if err != nil {
		t.Fatalf("stat database: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("database mode = %o, want 600", got)
	}

	var migrations int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 7 {
		t.Fatalf("migration count = %d, want 7", migrations)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations after rerun: %v", err)
	}
	if migrations != 7 {
		t.Fatalf("migration count after rerun = %d, want 7", migrations)
	}
}

func TestMigrateVersionFivePreservesExistingProxyAndCreditRows(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "v5.db"))
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
	} {
		version, err := migrationVersion(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(ctx, db, version, name); err != nil {
			t.Fatalf("apply old migration %s: %v", name, err)
		}
	}

	result, err := db.ExecContext(ctx, `
INSERT INTO proxy_users(username, desired_enabled, sync_state, created_at, updated_at)
VALUES ('legacy', 1, 'synced', 100, 100)`)
	if err != nil {
		t.Fatal(err)
	}
	proxyUserID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.ExecContext(ctx, `
INSERT INTO credit_buckets(
    proxy_user_id, original_bytes, consumed_bytes, starts_at, expires_at,
    reward_type, source, status, created_at, updated_at
) VALUES (?, 1024, 128, 100, NULL, 'legacy', 'fixture', 'active', 100, 100)`, proxyUserID)
	if err != nil {
		t.Fatal(err)
	}
	creditID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("upgrade from v5: %v", err)
	}
	var migrationCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 7 {
		t.Fatalf("migration count = %d, want 7", migrationCount)
	}
	var username string
	if err := db.QueryRowContext(ctx, "SELECT username FROM proxy_users WHERE id = ?", proxyUserID).Scan(&username); err != nil || username != "legacy" {
		t.Fatalf("legacy proxy user after upgrade = %q, %v", username, err)
	}
	var consumed int64
	var idempotencyKey sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT consumed_bytes, idempotency_key FROM credit_buckets WHERE id = ?", creditID).Scan(&consumed, &idempotencyKey); err != nil {
		t.Fatal(err)
	}
	if consumed != 128 || idempotencyKey.Valid {
		t.Fatalf("legacy credit changed during upgrade: consumed=%d idempotency=%#v", consumed, idempotencyKey)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO telegram_users(telegram_id, proxy_user_id, created_at, updated_at)
VALUES (12345, ?, 200, 200)`, proxyUserID); err != nil {
		t.Fatalf("new telegram_users table is unusable: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO settings(key, value, updated_at) VALUES ('start_gift_bytes', '100000000', 200)`); err != nil {
		t.Fatalf("new settings table is unusable: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO proxy_user_provisioning(
    proxy_user_id, phase, secret_sha256, last_error_code, created_at, updated_at
) VALUES (?, 'prepared', zeroblob(32), NULL, 200, 200)`, proxyUserID); err != nil {
		t.Fatalf("new proxy_user_provisioning table is unusable: %v", err)
	}
	var digestLength int
	if err := db.QueryRowContext(ctx, `SELECT length(secret_sha256) FROM proxy_user_provisioning WHERE proxy_user_id = ?`, proxyUserID).Scan(&digestLength); err != nil {
		t.Fatal(err)
	}
	if digestLength != 32 {
		t.Fatalf("provisioning digest length = %d, want 32", digestLength)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open(context.Background(), "   "); err == nil {
		t.Fatal("Open() error = nil, want error")
	}
}

func assertPragmaInt(t *testing.T, db interface {
	QueryRow(query string, args ...any) *sql.Row
}, name string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("PRAGMA " + name).Scan(&got); err != nil {
		t.Fatalf("read PRAGMA %s: %v", name, err)
	}
	if got != want {
		t.Fatalf("PRAGMA %s = %d, want %d", name, got, want)
	}
}
