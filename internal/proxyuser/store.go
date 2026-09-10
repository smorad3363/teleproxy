package proxyuser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("proxy user not found")
	ErrExists   = errors.New("proxy user already exists")
)

type SyncState string

const (
	SyncPending SyncState = "pending"
	SyncSynced  SyncState = "synced"
	SyncError   SyncState = "error"
)

type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	DesiredEnable bool      `json:"enabled"`
	SyncState     SyncState `json:"sync_state"`
	LastErrorCode string    `json:"last_error_code,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func Create(ctx context.Context, db *sql.DB, username string, enabled bool) (User, error) {
	if db == nil {
		return User{}, fmt.Errorf("proxy user database is required")
	}
	username = strings.TrimSpace(username)
	if err := ValidateUsername(username); err != nil {
		return User{}, err
	}
	if _, err := get(ctx, db, username); err == nil {
		return User{}, ErrExists
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, err
	}

	now := time.Now().UTC().Unix()
	result, err := db.ExecContext(ctx, `
INSERT INTO proxy_users(username, desired_enabled, sync_state, created_at, updated_at)
VALUES (?, ?, 'pending', ?, ?)`, username, boolInt(enabled), now, now)
	if err != nil {
		if _, lookupErr := get(ctx, db, username); lookupErr == nil {
			return User{}, ErrExists
		}
		return User{}, fmt.Errorf("create proxy user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("read proxy user id: %w", err)
	}
	return User{
		ID:            id,
		Username:      username,
		DesiredEnable: enabled,
		SyncState:     SyncPending,
		CreatedAt:     time.Unix(now, 0).UTC(),
		UpdatedAt:     time.Unix(now, 0).UTC(),
	}, nil
}

func List(ctx context.Context, db *sql.DB) ([]User, error) {
	if db == nil {
		return nil, fmt.Errorf("proxy user database is required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, username, desired_enabled, sync_state, last_error_code, created_at, updated_at
FROM proxy_users
ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list proxy users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy users: %w", err)
	}
	return users, nil
}

func SetDesiredEnabled(ctx context.Context, db *sql.DB, username string, enabled bool) (User, error) {
	if db == nil {
		return User{}, fmt.Errorf("proxy user database is required")
	}
	now := time.Now().UTC().Unix()
	result, err := db.ExecContext(ctx, `
UPDATE proxy_users
SET desired_enabled = ?, sync_state = 'pending', last_error_code = NULL, updated_at = ?
WHERE username = ?`, boolInt(enabled), now, username)
	if err != nil {
		return User{}, fmt.Errorf("update desired proxy user state: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("read proxy user update result: %w", err)
	}
	if changed == 0 {
		return User{}, ErrNotFound
	}
	return get(ctx, db, username)
}

func MarkSynced(ctx context.Context, db *sql.DB, username string) (User, error) {
	return markSync(ctx, db, username, SyncSynced, "")
}

func MarkSyncError(ctx context.Context, db *sql.DB, username, code string) (User, error) {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 64 {
		return User{}, fmt.Errorf("sync error code must be between 1 and 64 characters")
	}
	for _, r := range code {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return User{}, fmt.Errorf("sync error code contains unsupported characters")
	}
	return markSync(ctx, db, username, SyncError, code)
}

func ValidateUsername(username string) error {
	if len(username) < 1 || len(username) > 64 {
		return fmt.Errorf("proxy username must be between 1 and 64 characters")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return fmt.Errorf("proxy username contains unsupported characters")
	}
	return nil
}

func markSync(ctx context.Context, db *sql.DB, username string, state SyncState, code string) (User, error) {
	if db == nil {
		return User{}, fmt.Errorf("proxy user database is required")
	}
	now := time.Now().UTC().Unix()
	var result sql.Result
	var err error
	if code == "" {
		result, err = db.ExecContext(ctx, `
UPDATE proxy_users SET sync_state = ?, last_error_code = NULL, updated_at = ? WHERE username = ?`, state, now, username)
	} else {
		result, err = db.ExecContext(ctx, `
UPDATE proxy_users SET sync_state = ?, last_error_code = ?, updated_at = ? WHERE username = ?`, state, code, now, username)
	}
	if err != nil {
		return User{}, fmt.Errorf("mark proxy user sync state: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("read proxy user sync result: %w", err)
	}
	if changed == 0 {
		return User{}, ErrNotFound
	}
	return get(ctx, db, username)
}

func get(ctx context.Context, db *sql.DB, username string) (User, error) {
	row := db.QueryRowContext(ctx, `
SELECT id, username, desired_enabled, sync_state, last_error_code, created_at, updated_at
FROM proxy_users
WHERE username = ?
LIMIT 1`, username)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

type scanner interface {
	Scan(...any) error
}

func scanUser(row scanner) (User, error) {
	var user User
	var enabled int
	var state string
	var lastError sql.NullString
	var createdAt int64
	var updatedAt int64
	if err := row.Scan(&user.ID, &user.Username, &enabled, &state, &lastError, &createdAt, &updatedAt); err != nil {
		return User{}, err
	}
	user.DesiredEnable = enabled == 1
	user.SyncState = SyncState(state)
	if lastError.Valid {
		user.LastErrorCode = lastError.String
	}
	user.CreatedAt = time.Unix(createdAt, 0).UTC()
	user.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return user, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
