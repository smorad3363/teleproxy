package credit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	ErrReconciliationNotFound      = errors.New("quota reconciliation not found")
	ErrReconciliationConflict      = errors.New("quota reconciliation generation conflict")
	ErrReconciliationPhaseConflict = errors.New("quota reconciliation phase conflict")
	ErrReconciliationTransition    = errors.New("quota reconciliation phase transition is not allowed")
)

type ReconciliationPhase string

const (
	PhaseApplying  ReconciliationPhase = "applying"
	PhaseActive    ReconciliationPhase = "active"
	PhaseBlocking  ReconciliationPhase = "blocking"
	PhaseBlocked   ReconciliationPhase = "blocked"
	PhaseResetting ReconciliationPhase = "resetting"
	PhaseEnabling  ReconciliationPhase = "enabling"
)

func TransitionReconciliationPhase(ctx context.Context, db *sql.DB, username string, generation int64, from, to ReconciliationPhase) error {
	if db == nil {
		return fmt.Errorf("credit database is required")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return ErrUserNotFound
	}
	if generation <= 0 {
		return fmt.Errorf("reconciliation generation must be positive")
	}
	if !allowedReconciliationTransition(from, to) {
		return ErrReconciliationTransition
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reconciliation phase transition: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, err := proxyUserIDQuery(ctx, tx, username)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Truncate(time.Second).Unix()
	result, err := tx.ExecContext(ctx, `
UPDATE quota_reconciliations
SET phase = ?, updated_at = ?
WHERE proxy_user_id = ? AND generation = ? AND phase = ?`,
		string(to), now, userID, generation, string(from))
	if err != nil {
		return fmt.Errorf("transition reconciliation phase: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read reconciliation phase transition result: %w", err)
	}
	if changed != 1 {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM quota_reconciliations WHERE proxy_user_id = ?`, userID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReconciliationNotFound
		}
		if err != nil {
			return fmt.Errorf("inspect reconciliation phase conflict: %w", err)
		}
		return ErrReconciliationPhaseConflict
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reconciliation phase transition: %w", err)
	}
	return nil
}

func allowedReconciliationTransition(from, to ReconciliationPhase) bool {
	switch from {
	case PhaseActive:
		return to == PhaseBlocking
	case PhaseBlocking:
		return to == PhaseBlocked
	case PhaseBlocked:
		return to == PhaseResetting
	case PhaseApplying:
		return to == PhaseEnabling || to == PhaseActive
	case PhaseEnabling:
		return to == PhaseActive
	default:
		return false
	}
}

func knownReconciliationPhase(phase ReconciliationPhase) bool {
	switch phase {
	case PhaseApplying, PhaseActive, PhaseBlocking, PhaseBlocked, PhaseResetting, PhaseEnabling:
		return true
	default:
		return false
	}
}

type ProjectionMemberSnapshot struct {
	Position       int   `json:"position"`
	BucketID       int64 `json:"bucket_id"`
	AllowanceBytes int64 `json:"allowance_bytes"`
	AccountedBytes int64 `json:"accounted_bytes"`
}

type ReconciliationSnapshot struct {
	ProxyUserID             int64                      `json:"proxy_user_id"`
	Username                string                     `json:"username"`
	Generation              int64                      `json:"generation"`
	Phase                   ReconciliationPhase        `json:"phase"`
	TelemtResetEpochSecs    uint64                     `json:"telemt_reset_epoch_secs"`
	TelemtBaselineUsedBytes uint64                     `json:"telemt_baseline_used_bytes"`
	ProjectedAt             time.Time                  `json:"projected_at"`
	ProjectedQuotaBytes     uint64                     `json:"projected_quota_bytes"`
	EnforcedExpiry          *time.Time                 `json:"enforced_expiry,omitempty"`
	NextStart               *time.Time                 `json:"next_start,omitempty"`
	Members                 []ProjectionMemberSnapshot `json:"members"`
	UpdatedAt               time.Time                  `json:"updated_at"`
}

type PrepareProjectionInput struct {
	ExpectedGeneration      int64
	TelemtResetEpochSecs    uint64
	TelemtBaselineUsedBytes uint64
	ProjectedAt             time.Time
}

func PrepareProjection(ctx context.Context, db *sql.DB, username string, input PrepareProjectionInput) (ReconciliationSnapshot, error) {
	if db == nil {
		return ReconciliationSnapshot{}, fmt.Errorf("credit database is required")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return ReconciliationSnapshot{}, ErrUserNotFound
	}
	if input.ExpectedGeneration < 0 {
		return ReconciliationSnapshot{}, fmt.Errorf("expected generation must not be negative")
	}
	if input.ProjectedAt.IsZero() {
		return ReconciliationSnapshot{}, fmt.Errorf("projection time is required")
	}
	if input.TelemtResetEpochSecs > math.MaxInt64 {
		return ReconciliationSnapshot{}, fmt.Errorf("Telemt reset epoch exceeds SQLite signed integer range")
	}
	if input.TelemtBaselineUsedBytes > math.MaxInt64 {
		return ReconciliationSnapshot{}, fmt.Errorf("Telemt baseline usage exceeds SQLite signed integer range")
	}

	projectedAt := input.ProjectedAt.UTC().Truncate(time.Second)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("begin quota projection preparation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, err := proxyUserIDQuery(ctx, tx, username)
	if err != nil {
		return ReconciliationSnapshot{}, err
	}

	generation, err := nextReconciliationGeneration(ctx, tx, userID, input.ExpectedGeneration)
	if err != nil {
		return ReconciliationSnapshot{}, err
	}

	members, availableBytes, enforcedExpiry, nextStart, err := projectionMembers(ctx, tx, userID, projectedAt)
	if err != nil {
		return ReconciliationSnapshot{}, err
	}
	baseline := int64(input.TelemtBaselineUsedBytes)
	if availableBytes > math.MaxInt64-baseline {
		return ReconciliationSnapshot{}, fmt.Errorf("projected quota exceeds SQLite signed integer range")
	}
	projectedQuota := baseline + availableBytes

	if _, err := tx.ExecContext(ctx, "DELETE FROM quota_projection_members WHERE proxy_user_id = ?", userID); err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("clear previous quota projection members: %w", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	var enforcedExpiryUnix any
	if enforcedExpiry != nil {
		enforcedExpiryUnix = enforcedExpiry.Unix()
	}
	var nextStartUnix any
	if nextStart != nil {
		nextStartUnix = nextStart.Unix()
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO quota_reconciliations(
    proxy_user_id, generation, phase, telemt_reset_epoch_secs,
    telemt_baseline_used_bytes, projected_at, projected_quota_bytes,
    enforced_expiry, next_start, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(proxy_user_id) DO UPDATE SET
    generation = excluded.generation,
    phase = excluded.phase,
    telemt_reset_epoch_secs = excluded.telemt_reset_epoch_secs,
    telemt_baseline_used_bytes = excluded.telemt_baseline_used_bytes,
    projected_at = excluded.projected_at,
    projected_quota_bytes = excluded.projected_quota_bytes,
    enforced_expiry = excluded.enforced_expiry,
    next_start = excluded.next_start,
    updated_at = excluded.updated_at`,
		userID,
		generation,
		string(PhaseApplying),
		int64(input.TelemtResetEpochSecs),
		baseline,
		projectedAt.Unix(),
		projectedQuota,
		enforcedExpiryUnix,
		nextStartUnix,
		now.Unix(),
	)
	if err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("persist quota reconciliation: %w", err)
	}

	for _, member := range members {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO quota_projection_members(
    proxy_user_id, generation, position, bucket_id, allowance_bytes
) VALUES (?, ?, ?, ?, ?)`,
			userID, generation, member.Position, member.BucketID, member.AllowanceBytes,
		); err != nil {
			return ReconciliationSnapshot{}, fmt.Errorf("persist quota projection member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("commit quota projection preparation: %w", err)
	}

	return ReconciliationSnapshot{
		ProxyUserID:             userID,
		Username:                username,
		Generation:              generation,
		Phase:                   PhaseApplying,
		TelemtResetEpochSecs:    input.TelemtResetEpochSecs,
		TelemtBaselineUsedBytes: input.TelemtBaselineUsedBytes,
		ProjectedAt:             projectedAt,
		ProjectedQuotaBytes:     uint64(projectedQuota),
		EnforcedExpiry:          enforcedExpiry,
		NextStart:               nextStart,
		Members:                 members,
		UpdatedAt:               now,
	}, nil
}

func LoadReconciliation(ctx context.Context, db *sql.DB, username string) (ReconciliationSnapshot, error) {
	if db == nil {
		return ReconciliationSnapshot{}, fmt.Errorf("credit database is required")
	}
	username = strings.TrimSpace(username)
	userID, err := proxyUserID(ctx, db, username)
	if err != nil {
		return ReconciliationSnapshot{}, err
	}

	row := db.QueryRowContext(ctx, `
SELECT generation, phase, telemt_reset_epoch_secs, telemt_baseline_used_bytes,
       projected_at, projected_quota_bytes, enforced_expiry, next_start, updated_at
FROM quota_reconciliations
WHERE proxy_user_id = ?`, userID)

	var snapshot ReconciliationSnapshot
	var phase string
	var resetEpoch, baselineUsed, projectedAt, projectedQuota, updatedAt int64
	var enforcedExpiry, nextStart sql.NullInt64
	if err := row.Scan(
		&snapshot.Generation,
		&phase,
		&resetEpoch,
		&baselineUsed,
		&projectedAt,
		&projectedQuota,
		&enforcedExpiry,
		&nextStart,
		&updatedAt,
	); errors.Is(err, sql.ErrNoRows) {
		return ReconciliationSnapshot{}, ErrReconciliationNotFound
	} else if err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("load quota reconciliation: %w", err)
	}
	snapshot.Phase = ReconciliationPhase(phase)
	if snapshot.Generation <= 0 || !knownReconciliationPhase(snapshot.Phase) || resetEpoch < 0 || baselineUsed < 0 || projectedQuota < 0 {
		return ReconciliationSnapshot{}, fmt.Errorf("quota reconciliation contains invalid stored values")
	}

	snapshot.ProxyUserID = userID
	snapshot.Username = username
	snapshot.TelemtResetEpochSecs = uint64(resetEpoch)
	snapshot.TelemtBaselineUsedBytes = uint64(baselineUsed)
	snapshot.ProjectedAt = time.Unix(projectedAt, 0).UTC()
	snapshot.ProjectedQuotaBytes = uint64(projectedQuota)
	snapshot.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	if enforcedExpiry.Valid {
		value := time.Unix(enforcedExpiry.Int64, 0).UTC()
		snapshot.EnforcedExpiry = &value
	}
	if nextStart.Valid {
		value := time.Unix(nextStart.Int64, 0).UTC()
		snapshot.NextStart = &value
	}

	rows, err := db.QueryContext(ctx, `
SELECT position, bucket_id, allowance_bytes, accounted_bytes
FROM quota_projection_members
WHERE proxy_user_id = ? AND generation = ?
ORDER BY position`, userID, snapshot.Generation)
	if err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("load quota projection members: %w", err)
	}
	defer rows.Close()

	seenBuckets := make(map[int64]struct{})
	members := make([]ProjectionMemberSnapshot, 0)
	for rows.Next() {
		var member ProjectionMemberSnapshot
		if err := rows.Scan(&member.Position, &member.BucketID, &member.AllowanceBytes, &member.AccountedBytes); err != nil {
			return ReconciliationSnapshot{}, fmt.Errorf("scan quota projection member: %w", err)
		}
		if member.Position != len(members) || member.BucketID <= 0 || member.AllowanceBytes <= 0 || member.AccountedBytes < 0 || member.AccountedBytes > member.AllowanceBytes {
			return ReconciliationSnapshot{}, fmt.Errorf("quota projection member ordering or accounting is invalid")
		}
		if _, exists := seenBuckets[member.BucketID]; exists {
			return ReconciliationSnapshot{}, fmt.Errorf("quota projection contains duplicate bucket")
		}
		seenBuckets[member.BucketID] = struct{}{}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return ReconciliationSnapshot{}, fmt.Errorf("iterate quota projection members: %w", err)
	}
	snapshot.Members = members
	return snapshot, nil
}

func nextReconciliationGeneration(ctx context.Context, tx *sql.Tx, userID, expected int64) (int64, error) {
	var current int64
	var phase string
	err := tx.QueryRowContext(ctx, "SELECT generation, phase FROM quota_reconciliations WHERE proxy_user_id = ?", userID).Scan(&current, &phase)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if expected != 0 {
			return 0, ErrReconciliationConflict
		}
		return 1, nil
	case err != nil:
		return 0, fmt.Errorf("read quota reconciliation generation: %w", err)
	case current != expected:
		return 0, ErrReconciliationConflict
	case ReconciliationPhase(phase) != PhaseApplying && ReconciliationPhase(phase) != PhaseResetting:
		return 0, ErrReconciliationPhaseConflict
	case current == math.MaxInt64:
		return 0, fmt.Errorf("quota reconciliation generation exhausted")
	default:
		return current + 1, nil
	}
}

func projectionMembers(ctx context.Context, tx *sql.Tx, userID int64, projectedAt time.Time) ([]ProjectionMemberSnapshot, int64, *time.Time, *time.Time, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, original_bytes - consumed_bytes, starts_at, expires_at
FROM credit_buckets
WHERE proxy_user_id = ?
  AND status = 'active'
  AND consumed_bytes < original_bytes
ORDER BY CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END,
         expires_at ASC,
         starts_at ASC,
         id ASC`, userID)
	if err != nil {
		return nil, 0, nil, nil, fmt.Errorf("select credit buckets for projection: %w", err)
	}
	defer rows.Close()

	members := make([]ProjectionMemberSnapshot, 0)
	var available int64
	var enforcedExpiry *time.Time
	var nextStart *time.Time
	for rows.Next() {
		var bucketID, allowance, startsAt int64
		var expiresAt sql.NullInt64
		if err := rows.Scan(&bucketID, &allowance, &startsAt, &expiresAt); err != nil {
			return nil, 0, nil, nil, fmt.Errorf("scan credit bucket for projection: %w", err)
		}
		if bucketID <= 0 || allowance <= 0 {
			return nil, 0, nil, nil, fmt.Errorf("credit bucket contains invalid projection values")
		}
		starts := time.Unix(startsAt, 0).UTC()
		var expires *time.Time
		if expiresAt.Valid {
			value := time.Unix(expiresAt.Int64, 0).UTC()
			if !value.After(starts) {
				return nil, 0, nil, nil, fmt.Errorf("credit bucket contains invalid expiry")
			}
			expires = &value
		}

		if starts.After(projectedAt) {
			if nextStart == nil || starts.Before(*nextStart) {
				value := starts
				nextStart = &value
			}
			continue
		}
		if expires != nil && !expires.After(projectedAt) {
			continue
		}
		if available > math.MaxInt64-allowance {
			return nil, 0, nil, nil, fmt.Errorf("credit projection exceeds SQLite signed integer range")
		}
		available += allowance
		members = append(members, ProjectionMemberSnapshot{
			Position:       len(members),
			BucketID:       bucketID,
			AllowanceBytes: allowance,
		})
		if expires != nil && (enforcedExpiry == nil || expires.Before(*enforcedExpiry)) {
			value := *expires
			enforcedExpiry = &value
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, nil, nil, fmt.Errorf("iterate credit buckets for projection: %w", err)
	}
	return members, available, enforcedExpiry, nextStart, nil
}
