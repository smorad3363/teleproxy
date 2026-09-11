package admin

import (
	"context"
	"testing"
	"time"
)

func TestListInventoryReturnsNonSecretFieldsInIDOrder(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	owner, _, err := BootstrapOwner(ctx, db, "owner", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Unix(1_850_000_000, 0).UTC()
	updatedAt := createdAt.Add(time.Minute)
	result, err := db.ExecContext(ctx, `
		INSERT INTO admins(username, password_hash, role, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "support", "not-a-real-password-hash", "support", 0, createdAt.Unix(), updatedAt.Unix())
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	entries, err := ListInventory(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].ID != owner.ID || entries[0].Username != "owner" || entries[0].Role != "owner" || !entries[0].Enabled {
		t.Fatalf("owner entry = %#v", entries[0])
	}
	if entries[1].ID != secondID || entries[1].Username != "support" || entries[1].Role != "support" || entries[1].Enabled {
		t.Fatalf("second entry = %#v", entries[1])
	}
	if !entries[1].CreatedAt.Equal(createdAt) || !entries[1].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("second timestamps = %v %v", entries[1].CreatedAt, entries[1].UpdatedAt)
	}
}

func TestListInventoryReturnsEmptySlice(t *testing.T) {
	db := openTestDB(t)
	entries, err := ListInventory(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil || len(entries) != 0 {
		t.Fatalf("entries = %#v", entries)
	}
}
