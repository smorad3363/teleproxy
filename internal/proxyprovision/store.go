package proxyprovision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

var (
	ErrNotFound      = errors.New("proxy provisioning state not found")
	ErrStateConflict = errors.New("proxy provisioning state changed")
)

type Phase string

const (
	PhasePrepared  Phase = "prepared"
	PhaseOwned     Phase = "owned"
	PhaseCollision Phase = "collision"
)

type State struct {
	ProxyUserID   int64
	Username      string
	Phase         Phase
	SecretSHA256  [32]byte
	LastErrorCode string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func Prepare(ctx context.Context, db *sql.DB, username string, digest [32]byte, now time.Time) (State, bool, error) {
	if db == nil {
		return State{}, false, fmt.Errorf("proxy provisioning database is required")
	}
	username = strings.TrimSpace(username)
	if err := proxyuser.ValidateUsername(username); err != nil {
		return State{}, false, err
	}
	if now.IsZero() {
		return State{}, false, fmt.Errorf("proxy provisioning time is required")
	}
	now = now.UTC().Truncate(time.Second)

	var userID int64
	err := db.QueryRowContext(ctx, "SELECT id FROM proxy_users WHERE username = ?", username).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, false, proxyuser.ErrNotFound
	}
	if err != nil {
		return State{}, false, fmt.Errorf("find proxy user for provisioning: %w", err)
	}
	result, err := db.ExecContext(ctx, `
INSERT INTO proxy_user_provisioning(
    proxy_user_id, phase, secret_sha256, last_error_code, created_at, updated_at
) VALUES (?, 'prepared', ?, NULL, ?, ?)
ON CONFLICT(proxy_user_id) DO NOTHING`, userID, digest[:], now.Unix(), now.Unix())
	if err != nil {
		return State{}, false, fmt.Errorf("prepare proxy provisioning: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return State{}, false, fmt.Errorf("read proxy provisioning prepare result: %w", err)
	}
	state, err := Get(ctx, db, username)
	if err != nil {
		return State{}, false, err
	}
	return state, changed == 1, nil
}

func Get(ctx context.Context, db *sql.DB, username string) (State, error) {
	if db == nil {
		return State{}, fmt.Errorf("proxy provisioning database is required")
	}
	username = strings.TrimSpace(username)
	if err := proxyuser.ValidateUsername(username); err != nil {
		return State{}, err
	}
	row := db.QueryRowContext(ctx, `
SELECT pp.proxy_user_id, pu.username, pp.phase, pp.secret_sha256,
       pp.last_error_code, pp.created_at, pp.updated_at
FROM proxy_user_provisioning pp
JOIN proxy_users pu ON pu.id = pp.proxy_user_id
WHERE pu.username = ?`, username)
	return scanState(row)
}

func ReplacePreparedDigest(ctx context.Context, db *sql.DB, username string, expected, replacement [32]byte, now time.Time) (State, error) {
	if now.IsZero() {
		return State{}, fmt.Errorf("proxy provisioning time is required")
	}
	return updatePrepared(ctx, db, username, expected, now, `
UPDATE proxy_user_provisioning
SET secret_sha256 = ?, last_error_code = NULL, updated_at = ?
WHERE proxy_user_id = (SELECT id FROM proxy_users WHERE username = ?)
  AND phase = 'prepared'
  AND secret_sha256 = ?`, replacement[:], now.UTC().Truncate(time.Second).Unix(), strings.TrimSpace(username), expected[:])
}

func RecordError(ctx context.Context, db *sql.DB, username string, expected [32]byte, code string, now time.Time) (State, error) {
	if err := validateErrorCode(code); err != nil {
		return State{}, err
	}
	if now.IsZero() {
		return State{}, fmt.Errorf("proxy provisioning time is required")
	}
	return updatePrepared(ctx, db, username, expected, now, `
UPDATE proxy_user_provisioning
SET last_error_code = ?, updated_at = ?
WHERE proxy_user_id = (SELECT id FROM proxy_users WHERE username = ?)
  AND phase = 'prepared'
  AND secret_sha256 = ?`, code, now.UTC().Truncate(time.Second).Unix(), strings.TrimSpace(username), expected[:])
}

func MarkOwned(ctx context.Context, db *sql.DB, username string, proof OwnershipProof, now time.Time) (State, error) {
	return transitionPrepared(ctx, db, username, proof.digest, PhaseOwned, now)
}

func MarkCollision(ctx context.Context, db *sql.DB, username string, expected [32]byte, now time.Time) (State, error) {
	return transitionPrepared(ctx, db, username, expected, PhaseCollision, now)
}

func transitionPrepared(ctx context.Context, db *sql.DB, username string, expected [32]byte, target Phase, now time.Time) (State, error) {
	if target != PhaseOwned && target != PhaseCollision {
		return State{}, fmt.Errorf("unsupported proxy provisioning transition")
	}
	if now.IsZero() {
		return State{}, fmt.Errorf("proxy provisioning time is required")
	}
	username = strings.TrimSpace(username)
	if err := proxyuser.ValidateUsername(username); err != nil {
		return State{}, err
	}
	if db == nil {
		return State{}, fmt.Errorf("proxy provisioning database is required")
	}
	result, err := db.ExecContext(ctx, `
UPDATE proxy_user_provisioning
SET phase = ?, last_error_code = NULL, updated_at = ?
WHERE proxy_user_id = (SELECT id FROM proxy_users WHERE username = ?)
  AND phase = 'prepared'
  AND secret_sha256 = ?`, target, now.UTC().Truncate(time.Second).Unix(), username, expected[:])
	if err != nil {
		return State{}, fmt.Errorf("transition proxy provisioning: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return State{}, fmt.Errorf("read proxy provisioning transition result: %w", err)
	}
	state, getErr := Get(ctx, db, username)
	if getErr != nil {
		return State{}, getErr
	}
	if changed == 1 {
		return state, nil
	}
	if state.Phase == target && state.SecretSHA256 == expected {
		return state, nil
	}
	return State{}, ErrStateConflict
}

func updatePrepared(ctx context.Context, db *sql.DB, username string, expected [32]byte, now time.Time, statement string, args ...any) (State, error) {
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
	result, err := db.ExecContext(ctx, statement, args...)
	if err != nil {
		return State{}, fmt.Errorf("update prepared proxy provisioning: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return State{}, fmt.Errorf("read prepared proxy provisioning update result: %w", err)
	}
	if changed != 1 {
		return State{}, ErrStateConflict
	}
	return Get(ctx, db, username)
}

func validateErrorCode(code string) error {
	if code == "" || len(code) > 64 {
		return fmt.Errorf("proxy provisioning error code must be between 1 and 64 characters")
	}
	for _, ch := range code {
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return fmt.Errorf("proxy provisioning error code contains unsupported characters")
	}
	return nil
}

type scanner interface {
	Scan(...any) error
}

func scanState(row scanner) (State, error) {
	var state State
	var phase string
	var digest []byte
	var lastError sql.NullString
	var createdAt, updatedAt int64
	if err := row.Scan(&state.ProxyUserID, &state.Username, &phase, &digest, &lastError, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return State{}, ErrNotFound
		}
		return State{}, fmt.Errorf("read proxy provisioning state: %w", err)
	}
	if len(digest) != len(state.SecretSHA256) {
		return State{}, fmt.Errorf("proxy provisioning secret digest is invalid")
	}
	copy(state.SecretSHA256[:], digest)
	switch Phase(phase) {
	case PhasePrepared, PhaseOwned, PhaseCollision:
		state.Phase = Phase(phase)
	default:
		return State{}, fmt.Errorf("proxy provisioning phase is invalid")
	}
	if lastError.Valid {
		if err := validateErrorCode(lastError.String); err != nil {
			return State{}, fmt.Errorf("proxy provisioning error code is invalid")
		}
		state.LastErrorCode = lastError.String
	}
	state.CreatedAt = time.Unix(createdAt, 0).UTC()
	state.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return state, nil
}
