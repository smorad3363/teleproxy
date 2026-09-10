package admin

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/smorad3363/teleproxy/internal/auth"
	"github.com/smorad3363/teleproxy/internal/database"
)

func TestBootstrapOwnerIsIdempotentAndDoesNotStorePlaintext(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	password := "generated-admin-password-123"
	first, created, err := BootstrapOwner(ctx, db, "admin", password)
	if err != nil {
		t.Fatalf("BootstrapOwner(first) error = %v", err)
	}
	if !created {
		t.Fatal("BootstrapOwner(first) created = false")
	}

	second, created, err := BootstrapOwner(ctx, db, "another-admin", "another-password-long-enough")
	if err != nil {
		t.Fatalf("BootstrapOwner(second) error = %v", err)
	}
	if created {
		t.Fatal("BootstrapOwner(second) created = true")
	}
	if second.ID != first.ID || second.Username != first.Username {
		t.Fatalf("second bootstrap returned a different admin: %#v vs %#v", second, first)
	}

	var storedHash string
	if err := db.QueryRow("SELECT password_hash FROM admins WHERE id = ?", first.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read stored password hash: %v", err)
	}
	if storedHash == password {
		t.Fatal("database stored plaintext password")
	}
	if !auth.VerifyPassword(storedHash, password) {
		t.Fatal("stored password hash does not verify")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count); err != nil {
		t.Fatalf("count admins: %v", err)
	}
	if count != 1 {
		t.Fatalf("admin count = %d, want 1", count)
	}
}

func TestBootstrapOwnerRejectsWeakBootstrapInput(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, _, err := BootstrapOwner(ctx, db, "a", "generated-admin-password-123"); err == nil {
		t.Fatal("invalid username accepted")
	}
	if _, _, err := BootstrapOwner(ctx, db, "admin", "short"); err == nil {
		t.Fatal("short bootstrap password accepted")
	}
}
