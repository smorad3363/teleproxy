package settings

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestStartGiftDefaultsAndCanBeChanged(t *testing.T) {
	db := settingsTestDB(t)
	got, err := StartGiftBytes(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if got != DefaultStartGiftBytes {
		t.Fatalf("default start gift = %d, want %d", got, DefaultStartGiftBytes)
	}
	if err := SetStartGiftBytes(context.Background(), db, 250_000_000); err != nil {
		t.Fatal(err)
	}
	got, err = StartGiftBytes(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if got != 250_000_000 {
		t.Fatalf("configured start gift = %d", got)
	}
}

func TestStartGiftRejectsInvalidValuesAndCorruptStorage(t *testing.T) {
	db := settingsTestDB(t)
	if err := SetStartGiftBytes(context.Background(), db, 0); err == nil {
		t.Fatal("SetStartGiftBytes accepted zero")
	}
	if _, err := db.Exec(`INSERT INTO settings(key, value, updated_at) VALUES ('start_gift_bytes', 'invalid', 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := StartGiftBytes(context.Background(), db); err == nil {
		t.Fatal("StartGiftBytes accepted corrupt stored value")
	}
}

func settingsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
