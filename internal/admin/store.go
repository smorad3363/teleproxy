package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/auth"
)

type Admin struct {
	ID       int64
	Username string
	Role     string
	Enabled  bool
}

func BootstrapOwner(ctx context.Context, db *sql.DB, username, password string) (Admin, bool, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return Admin{}, false, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Admin{}, false, fmt.Errorf("begin admin bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := firstAdmin(ctx, tx)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Admin{}, false, fmt.Errorf("check existing admin: %w", err)
	}

	if len(password) < 16 {
		return Admin{}, false, fmt.Errorf("bootstrap password must be at least 16 bytes")
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return Admin{}, false, fmt.Errorf("hash bootstrap password: %w", err)
	}

	now := time.Now().UTC().Unix()
	result, err := tx.ExecContext(ctx, `
INSERT INTO admins(username, password_hash, role, enabled, created_at, updated_at)
VALUES (?, ?, 'owner', 1, ?, ?)`, username, passwordHash, now, now)
	if err != nil {
		return Admin{}, false, fmt.Errorf("insert bootstrap admin: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Admin{}, false, fmt.Errorf("read bootstrap admin id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Admin{}, false, fmt.Errorf("commit admin bootstrap: %w", err)
	}

	return Admin{ID: id, Username: username, Role: "owner", Enabled: true}, true, nil
}

func firstAdmin(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (Admin, error) {
	var admin Admin
	var enabled int
	err := q.QueryRowContext(ctx, `
SELECT id, username, role, enabled
FROM admins
ORDER BY id
LIMIT 1`).Scan(&admin.ID, &admin.Username, &admin.Role, &enabled)
	if err != nil {
		return Admin{}, err
	}
	admin.Enabled = enabled == 1
	return admin, nil
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 64 {
		return fmt.Errorf("admin username must be between 3 and 64 characters")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return fmt.Errorf("admin username contains unsupported characters")
	}
	return nil
}
