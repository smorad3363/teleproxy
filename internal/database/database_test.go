package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenAppliesSQLiteInvariantsAndMigrations(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
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

	var migrations int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 1 {
		t.Fatalf("migration count = %d, want 1", migrations)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations after rerun: %v", err)
	}
	if migrations != 1 {
		t.Fatalf("migration count after rerun = %d, want 1", migrations)
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
