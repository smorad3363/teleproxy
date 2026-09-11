package auditlog

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

func TestAppendGetListRoundTripsAndOrders(t *testing.T) {
	ctx := context.Background()
	db := auditTestDB(t)
	before := `{"enabled":false,"label":"قبل"}`
	after := `{"enabled":true,"label":"بعد"}`
	first, err := Append(ctx, db, AppendInput{
		Actor: "admin:owner", Action: "user.enable", Target: "proxy_user:42",
		BeforeSnapshot: &before, AfterSnapshot: &after, RequestID: "req-001",
	}, time.Unix(1_840_000_000, 987654321).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID <= 0 || first.BeforeSnapshot == nil || *first.BeforeSnapshot != before || first.AfterSnapshot == nil || *first.AfterSnapshot != after {
		t.Fatalf("first entry = %#v", first)
	}
	if !first.CreatedAt.Equal(time.Unix(1_840_000_000, 0).UTC()) {
		t.Fatalf("created_at = %v", first.CreatedAt)
	}

	second, err := Append(ctx, db, AppendInput{
		Actor: "system", Action: "node.observe", Target: "node:7", RequestID: "req-002",
	}, time.Unix(1_840_000_001, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	got, err := Get(ctx, db, first.ID)
	if err != nil || got.RequestID != first.RequestID || got.Actor != first.Actor {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
	entries, err := List(ctx, db, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].ID != second.ID || entries[1].ID != first.ID {
		t.Fatalf("List() = %#v", entries)
	}
	older, err := List(ctx, db, second.ID, 10)
	if err != nil || len(older) != 1 || older[0].ID != first.ID {
		t.Fatalf("cursor List() = %#v, %v", older, err)
	}
}

func TestListEmptyIsNonNilAndBounded(t *testing.T) {
	db := auditTestDB(t)
	entries, err := List(context.Background(), db, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil || len(entries) != 0 {
		t.Fatalf("empty List() = %#v", entries)
	}
	for _, input := range []struct {
		before int64
		limit  int
	}{{-1, 1}, {0, 0}, {0, 101}} {
		if _, err := List(context.Background(), db, input.before, input.limit); !errors.Is(err, ErrInvalid) {
			t.Fatalf("List(%d,%d) error = %v, want ErrInvalid", input.before, input.limit, err)
		}
	}
}

func TestAppendRejectsInvalidInputWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := auditTestDB(t)
	now := time.Unix(1_840_001_000, 0).UTC()
	invalidUTF8 := string([]byte{0xff})
	tooLarge := strings.Repeat("x", maxSnapshotBytes+1)

	tests := []AppendInput{
		{Actor: "", Action: "x", Target: "y", RequestID: "r"},
		{Actor: " actor ", Action: "x", Target: "y", RequestID: "r"},
		{Actor: invalidUTF8, Action: "x", Target: "y", RequestID: "r"},
		{Actor: "actor", Action: invalidUTF8, Target: "y", RequestID: "r"},
		{Actor: "actor", Action: "x", Target: invalidUTF8, RequestID: "r"},
		{Actor: "actor", Action: "x", Target: "y", RequestID: invalidUTF8},
		{Actor: "actor", Action: "x", Target: "y", RequestID: "r", BeforeSnapshot: &invalidUTF8},
		{Actor: "actor", Action: "x", Target: "y", RequestID: "r", AfterSnapshot: &tooLarge},
	}
	for index, input := range tests {
		if _, err := Append(ctx, db, input, now.Add(time.Duration(index)*time.Second)); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Append(%d) error = %v, want ErrInvalid", index, err)
		}
	}
	if _, err := Append(ctx, db, AppendInput{Actor: "actor", Action: "x", Target: "y", RequestID: "r"}, time.Time{}); err == nil {
		t.Fatal("zero-time Append() unexpectedly succeeded")
	}
	if got := countAuditRows(t, db); got != 0 {
		t.Fatalf("invalid appends mutated audit log: %d rows", got)
	}
}

func TestAuditLogIsDatabaseAppendOnly(t *testing.T) {
	ctx := context.Background()
	db := auditTestDB(t)
	entry, err := Append(ctx, db, AppendInput{Actor: "actor", Action: "x", Target: "y", RequestID: "req"}, time.Unix(1_840_002_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE audit_log SET action = 'changed' WHERE id = ?", entry.ID); err == nil {
		t.Fatal("audit log UPDATE unexpectedly succeeded")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM audit_log WHERE id = ?", entry.ID); err == nil {
		t.Fatal("audit log DELETE unexpectedly succeeded")
	}
	if got := countAuditRows(t, db); got != 1 {
		t.Fatalf("append-only row count = %d, want 1", got)
	}
}

func TestAppendDoesNotImplicitlyCopySensitiveState(t *testing.T) {
	ctx := context.Background()
	db := auditTestDB(t)
	const marker = "raw-secret-marker-do-not-copy"
	if _, err := db.ExecContext(ctx, `
INSERT INTO admins(username, password_hash, role, enabled, created_at, updated_at)
VALUES ('owner', ?, 'owner', 1, 1, 1)`, marker); err != nil {
		t.Fatal(err)
	}
	if _, err := Append(ctx, db, AppendInput{Actor: "admin:owner", Action: "settings.view", Target: "settings", RequestID: "req-safe"}, time.Unix(1_840_003_000, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	var combined string
	if err := db.QueryRowContext(ctx, `
SELECT actor || action || target || COALESCE(before_snapshot, '') || COALESCE(after_snapshot, '') || request_id
FROM audit_log
LIMIT 1`).Scan(&combined); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(combined, marker) {
		t.Fatal("audit log implicitly copied sensitive database state")
	}
}

func auditTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countAuditRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
