package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/auth"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
)

type Admin struct {
	ID       int64
	Username string
	Role     string
	Enabled  bool
}

type Session struct {
	Admin     Admin
	ExpiresAt time.Time
}

func BootstrapOwnerFromFile(ctx context.Context, db *sql.DB, username, passwordFile string) (Admin, bool, error) {
	existing, err := firstAdmin(ctx, db)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Admin{}, false, fmt.Errorf("check existing admin: %w", err)
	}
	if passwordFile == "" {
		return Admin{}, false, fmt.Errorf("bootstrap password file is required when no admin exists")
	}

	info, err := os.Stat(passwordFile)
	if err != nil {
		return Admin{}, false, fmt.Errorf("stat bootstrap password file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Admin{}, false, fmt.Errorf("bootstrap password file must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return Admin{}, false, fmt.Errorf("bootstrap password file must not be accessible by group or others")
	}
	if info.Size() > 4096 {
		return Admin{}, false, fmt.Errorf("bootstrap password file is too large")
	}

	data, err := os.ReadFile(passwordFile)
	if err != nil {
		return Admin{}, false, fmt.Errorf("read bootstrap password file: %w", err)
	}
	password := strings.TrimRight(string(data), "\r\n")
	return BootstrapOwner(ctx, db, username, password)
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

func Authenticate(ctx context.Context, db *sql.DB, username, password string) (Admin, error) {
	var admin Admin
	var passwordHash string
	var enabled int
	err := db.QueryRowContext(ctx, `
SELECT id, username, password_hash, role, enabled
FROM admins
WHERE username = ?
LIMIT 1`, strings.TrimSpace(username)).Scan(
		&admin.ID, &admin.Username, &passwordHash, &admin.Role, &enabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrInvalidCredentials
	}
	if err != nil {
		return Admin{}, fmt.Errorf("query admin credentials: %w", err)
	}
	admin.Enabled = enabled == 1
	if !admin.Enabled || !auth.VerifyPassword(passwordHash, password) {
		return Admin{}, ErrInvalidCredentials
	}
	return admin, nil
}

func CreateSession(ctx context.Context, db *sql.DB, adminID int64, ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 {
		return "", time.Time{}, fmt.Errorf("session TTL must be positive")
	}
	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	if _, err := db.ExecContext(ctx, `
INSERT INTO admin_sessions(token_hash, admin_id, expires_at, created_at)
VALUES (?, ?, ?, ?)`, tokenHash, adminID, expiresAt.Unix(), now.Unix()); err != nil {
		return "", time.Time{}, fmt.Errorf("create admin session: %w", err)
	}
	return token, expiresAt, nil
}

func SessionByToken(ctx context.Context, db *sql.DB, token string) (Session, error) {
	tokenHash, err := auth.HashSessionToken(token)
	if err != nil {
		return Session{}, ErrSessionNotFound
	}

	var session Session
	var expiresAt int64
	var enabled int
	err = db.QueryRowContext(ctx, `
SELECT a.id, a.username, a.role, a.enabled, s.expires_at
FROM admin_sessions AS s
JOIN admins AS a ON a.id = s.admin_id
WHERE s.token_hash = ?
LIMIT 1`, tokenHash).Scan(
		&session.Admin.ID,
		&session.Admin.Username,
		&session.Admin.Role,
		&enabled,
		&expiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("query admin session: %w", err)
	}
	session.Admin.Enabled = enabled == 1
	session.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	if !session.Admin.Enabled || !session.ExpiresAt.After(time.Now().UTC()) {
		_, _ = db.ExecContext(ctx, "DELETE FROM admin_sessions WHERE token_hash = ?", tokenHash)
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func RevokeSession(ctx context.Context, db *sql.DB, token string) error {
	tokenHash, err := auth.HashSessionToken(token)
	if err != nil {
		return nil
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM admin_sessions WHERE token_hash = ?", tokenHash); err != nil {
		return fmt.Errorf("revoke admin session: %w", err)
	}
	return nil
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
