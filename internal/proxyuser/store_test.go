package proxyuser

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestProxyUserLifecycleState(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	created, err := Create(context.Background(), db, "alice", false)
	if err != nil {
		t.Fatal(err)
	}
	if created.DesiredEnable || created.SyncState != SyncPending {
		t.Fatalf("created = %#v", created)
	}

	failed, err := MarkSyncError(context.Background(), db, "alice", "TELEMT_UNAVAILABLE")
	if err != nil {
		t.Fatal(err)
	}
	if failed.SyncState != SyncError || failed.LastErrorCode != "TELEMT_UNAVAILABLE" {
		t.Fatalf("failed = %#v", failed)
	}

	pending, err := SetDesiredEnabled(context.Background(), db, "alice", true)
	if err != nil {
		t.Fatal(err)
	}
	if !pending.DesiredEnable || pending.SyncState != SyncPending || pending.LastErrorCode != "" {
		t.Fatalf("pending = %#v", pending)
	}

	synced, err := MarkSynced(context.Background(), db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if synced.SyncState != SyncSynced || synced.LastErrorCode != "" {
		t.Fatalf("synced = %#v", synced)
	}
}

func TestProxyUserListAndErrors(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := Create(context.Background(), db, "bad user", true); err == nil {
		t.Fatal("Create() accepted invalid username")
	}
	if _, err := Create(context.Background(), db, "bob", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(context.Background(), db, "alice", false); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(context.Background(), db, "alice", true); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate Create() error = %v", err)
	}
	users, err := List(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Username != "alice" || users[1].Username != "bob" {
		t.Fatalf("List() = %#v", users)
	}
	if _, err := SetDesiredEnabled(context.Background(), db, "missing", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing SetDesiredEnabled() error = %v", err)
	}
}

func TestProxyUserSchemaContainsNoSecretColumn(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`PRAGMA table_info(proxy_users)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(name), "secret") {
			t.Fatalf("proxy_users contains secret-bearing column %q", name)
		}
	}
}
