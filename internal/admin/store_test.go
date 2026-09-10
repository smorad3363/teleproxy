package admin

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/auth"
	"github.com/smorad3363/teleproxy/internal/database"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestBootstrapOwnerIsIdempotentAndDoesNotStorePlaintext(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

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
	if created || second.ID != first.ID || second.Username != first.Username {
		t.Fatalf("second bootstrap changed admin: %#v, created=%v", second, created)
	}

	var storedHash string
	if err := db.QueryRow("SELECT password_hash FROM admins WHERE id = ?", first.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read stored password hash: %v", err)
	}
	if storedHash == password || !auth.VerifyPassword(storedHash, password) {
		t.Fatal("stored password is plaintext or does not verify")
	}
}

func TestBootstrapOwnerFromFileRequiresProtectedFileOnlyForFirstAdmin(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	passwordFile := filepath.Join(t.TempDir(), "admin-password")
	if err := os.WriteFile(passwordFile, []byte("generated-admin-password-123\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	first, created, err := BootstrapOwnerFromFile(ctx, db, "admin", passwordFile)
	if err != nil || !created {
		t.Fatalf("BootstrapOwnerFromFile(first) = %#v, %v, %v", first, created, err)
	}
	if err := os.Remove(passwordFile); err != nil {
		t.Fatal(err)
	}
	second, created, err := BootstrapOwnerFromFile(ctx, db, "admin", passwordFile)
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("restart bootstrap = %#v, %v, %v", second, created, err)
	}
}

func TestBootstrapOwnerFromFileRejectsBroadPermissions(t *testing.T) {
	db := openTestDB(t)
	passwordFile := filepath.Join(t.TempDir(), "admin-password")
	if err := os.WriteFile(passwordFile, []byte("generated-admin-password-123"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := BootstrapOwnerFromFile(context.Background(), db, "admin", passwordFile); err == nil {
		t.Fatal("broad bootstrap file permissions accepted")
	}
}

func TestAuthenticateAndSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	owner, _, err := BootstrapOwner(ctx, db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Authenticate(ctx, db, "admin", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v", err)
	}
	authenticated, err := Authenticate(ctx, db, "admin", "generated-admin-password-123")
	if err != nil || authenticated.ID != owner.ID {
		t.Fatalf("Authenticate() = %#v, %v", authenticated, err)
	}

	token, expiresAt, err := CreateSession(ctx, db, owner.ID, time.Hour)
	if err != nil || token == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("CreateSession() token=%q expires=%v err=%v", token, expiresAt, err)
	}

	var stored []byte
	if err := db.QueryRow("SELECT token_hash FROM admin_sessions").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if string(stored) == token {
		t.Fatal("session token stored plaintext")
	}

	session, err := SessionByToken(ctx, db, token)
	if err != nil || session.Admin.ID != owner.ID {
		t.Fatalf("SessionByToken() = %#v, %v", session, err)
	}
	if err := RevokeSession(ctx, db, token); err != nil {
		t.Fatal(err)
	}
	if _, err := SessionByToken(ctx, db, token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("revoked session error = %v", err)
	}
}

func TestBootstrapOwnerRejectsWeakBootstrapInput(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if _, _, err := BootstrapOwner(ctx, db, "a", "generated-admin-password-123"); err == nil {
		t.Fatal("invalid username accepted")
	}
	if _, _, err := BootstrapOwner(ctx, db, "admin", "short"); err == nil {
		t.Fatal("short bootstrap password accepted")
	}
}
