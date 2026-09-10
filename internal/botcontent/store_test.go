package botcontent

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestSetGetListClearBotContent(t *testing.T) {
	ctx := context.Background()
	db := botContentTestDB(t)
	createdAt := time.Unix(1_840_000_000, 900_000_000).UTC()

	welcome, err := Set(ctx, db, SlotWelcome, "سلام 👋", createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if welcome.Slot != SlotWelcome || welcome.Text != "سلام 👋" || !welcome.CreatedAt.Equal(createdAt.Truncate(time.Second)) || !welcome.UpdatedAt.Equal(welcome.CreatedAt) {
		t.Fatalf("welcome = %#v", welcome)
	}
	if _, err := Set(ctx, db, SlotSupport, "Support text", createdAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	updatedAt := createdAt.Add(2 * time.Second)
	updated, err := Set(ctx, db, SlotWelcome, "Welcome back", updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Text != "Welcome back" || !updated.CreatedAt.Equal(welcome.CreatedAt) || !updated.UpdatedAt.Equal(updatedAt.Truncate(time.Second)) {
		t.Fatalf("updated welcome = %#v", updated)
	}

	got, err := Get(ctx, db, SlotWelcome)
	if err != nil || got != updated {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
	entries, err := List(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Slot != SlotSupport || entries[1].Slot != SlotWelcome {
		t.Fatalf("List() = %#v", entries)
	}

	if err := Clear(ctx, db, SlotSupport); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, db, SlotSupport); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(cleared) error = %v, want ErrNotFound", err)
	}
	if err := Clear(ctx, db, SlotSupport); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Clear(cleared) error = %v, want ErrNotFound", err)
	}
	entries, err = List(ctx, db)
	if err != nil || len(entries) != 1 || entries[0].Slot != SlotWelcome {
		t.Fatalf("List() after clear = %#v, %v", entries, err)
	}
}

func TestBotContentAcceptsEveryRoadmapTextSlot(t *testing.T) {
	ctx := context.Background()
	db := botContentTestDB(t)
	now := time.Unix(1_840_001_000, 0).UTC()
	slots := []Slot{SlotWelcome, SlotForcedJoin, SlotReferral, SlotProxy, SlotExpired, SlotNoCredit, SlotSupport}
	for index, slot := range slots {
		if _, err := Set(ctx, db, slot, "configured "+string(slot), now.Add(time.Duration(index)*time.Second)); err != nil {
			t.Fatalf("Set(%q) = %v", slot, err)
		}
	}
	if got := countBotContent(t, db); got != len(slots) {
		t.Fatalf("configured slot count = %d, want %d", got, len(slots))
	}
}

func TestBotContentRejectsInvalidInputWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := botContentTestDB(t)
	now := time.Unix(1_840_002_000, 0).UTC()
	invalidUTF8 := string([]byte{0xff})

	tests := []struct {
		name string
		slot Slot
		text string
	}{
		{name: "unknown slot", slot: Slot("other"), text: "ok"},
		{name: "empty text", slot: SlotWelcome, text: ""},
		{name: "invalid UTF-8", slot: SlotWelcome, text: invalidUTF8},
		{name: "too long", slot: SlotWelcome, text: strings.Repeat("🙂", MaxTextRunes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Set(ctx, db, test.slot, test.text, now); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Set() error = %v, want ErrInvalid", err)
			}
		})
	}
	if countBotContent(t, db) != 0 {
		t.Fatal("invalid bot content mutated table")
	}
	if _, err := Get(ctx, db, Slot("other")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Get(invalid slot) error = %v, want ErrInvalid", err)
	}
	if err := Clear(ctx, db, Slot("other")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Clear(invalid slot) error = %v, want ErrInvalid", err)
	}
}

func TestBotContentSchemaRejectsUnknownSlotAndOverlongText(t *testing.T) {
	db := botContentTestDB(t)
	if _, err := db.Exec(`
INSERT INTO bot_content(slot, text, created_at, updated_at)
VALUES ('unknown', 'text', 1, 1)`); err == nil {
		t.Fatal("schema accepted unknown bot content slot")
	}
	if _, err := db.Exec(`
INSERT INTO bot_content(slot, text, created_at, updated_at)
VALUES ('welcome', ?, 1, 1)`, strings.Repeat("x", MaxTextRunes+1)); err == nil {
		t.Fatal("schema accepted overlong bot content text")
	}
	if countBotContent(t, db) != 0 {
		t.Fatal("invalid direct inserts mutated bot content table")
	}
}

func botContentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countBotContent(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM bot_content").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
