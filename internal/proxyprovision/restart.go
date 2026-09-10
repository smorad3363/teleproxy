package proxyprovision

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

func RestartOwned(ctx context.Context, db *sql.DB, username string, expected, replacement [32]byte, now time.Time) (State, error) {
	if db == nil {
		return State{}, fmt.Errorf("proxy provisioning database is required")
	}
	username = strings.TrimSpace(username)
	if err := proxyuser.ValidateUsername(username); err != nil {
		return State{}, err
	}
	if now.IsZero() {
		return State{}, fmt.Errorf("proxy provisioning time is required")
	}
	result, err := db.ExecContext(ctx, `
UPDATE proxy_user_provisioning
SET phase = 'prepared', secret_sha256 = ?, last_error_code = NULL, updated_at = ?
WHERE proxy_user_id = (SELECT id FROM proxy_users WHERE username = ?)
  AND phase = 'owned'
  AND secret_sha256 = ?`, replacement[:], now.UTC().Truncate(time.Second).Unix(), username, expected[:])
	if err != nil {
		return State{}, fmt.Errorf("restart owned proxy provisioning: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return State{}, fmt.Errorf("read proxy provisioning restart result: %w", err)
	}
	if changed != 1 {
		return State{}, ErrStateConflict
	}
	return Get(ctx, db, username)
}
