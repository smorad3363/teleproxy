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
	ErrReconciliationNotActive         = errors.New("quota reconciliation is not active")
	ErrQuotaResetMismatch              = errors.New("Telemt quota reset epoch does not match reconciliation")
	ErrQuotaUsageRegressed             = errors.New("Telemt quota usage regressed")
	ErrProjectionDrift                 = errors.New("credit projection drift detected")
	ErrInsufficientProjectionAllowance = errors.New("projection allowance is insufficient for observed usage")
)

type QuotaUsageObservation struct {
	ResetEpochSecs uint64
	UsedBytes      uint64
}

type UsageAccounting struct {
	Generation        int64        `json:"generation"`
	PreviousUsedBytes uint64       `json:"previous_used_bytes"`
	ObservedUsedBytes uint64       `json:"observed_used_bytes"`
	AccountedBytes    int64        `json:"accounted_bytes"`
	Allocations       []Allocation `json:"allocations"`
}

func AccountObservedUsage(ctx context.Context, db *sql.DB, username string, observation QuotaUsageObservation) (UsageAccounting, error) {
	if db == nil {
		return UsageAccounting{}, fmt.Errorf("credit database is required")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return UsageAccounting{}, ErrUserNotFound
	}
	if observation.ResetEpochSecs > math.MaxInt64 {
		return UsageAccounting{}, fmt.Errorf("Telemt reset epoch exceeds SQLite signed integer range")
	}
	if observation.UsedBytes > math.MaxInt64 {
		return UsageAccounting{}, fmt.Errorf("Telemt usage exceeds SQLite signed integer range")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return UsageAccounting{}, fmt.Errorf("begin quota usage accounting: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, err := proxyUserIDQuery(ctx, tx, username)
	if err != nil {
		return UsageAccounting{}, err
	}

	var generation int64
	var phase string
	var resetEpoch, baselineUsed int64
	err = tx.QueryRowContext(ctx, `
SELECT generation, phase, telemt_reset_epoch_secs, telemt_baseline_used_bytes
FROM quota_reconciliations
WHERE proxy_user_id = ?`, userID).Scan(&generation, &phase, &resetEpoch, &baselineUsed)
	if errors.Is(err, sql.ErrNoRows) {
		return UsageAccounting{}, ErrReconciliationNotFound
	}
	if err != nil {
		return UsageAccounting{}, fmt.Errorf("load quota reconciliation for accounting: %w", err)
	}
	if generation <= 0 || resetEpoch < 0 || baselineUsed < 0 {
		return UsageAccounting{}, ErrProjectionDrift
	}
	if ReconciliationPhase(phase) != PhaseActive {
		return UsageAccounting{}, ErrReconciliationNotActive
	}
	if uint64(resetEpoch) != observation.ResetEpochSecs {
		return UsageAccounting{}, ErrQuotaResetMismatch
	}
	if observation.UsedBytes < uint64(baselineUsed) {
		return UsageAccounting{}, ErrQuotaUsageRegressed
	}

	delta := int64(observation.UsedBytes - uint64(baselineUsed))
	result := UsageAccounting{
		Generation:        generation,
		PreviousUsedBytes: uint64(baselineUsed),
		ObservedUsedBytes: observation.UsedBytes,
		AccountedBytes:    delta,
		Allocations:       []Allocation{},
	}

	members, err := loadAccountingMembers(ctx, tx, userID, generation)
	if err != nil {
		return UsageAccounting{}, err
	}
	var remainingAllowance int64
	for _, member := range members {
		available := member.allowanceBytes - member.accountedBytes
		if remainingAllowance > math.MaxInt64-available {
			return UsageAccounting{}, ErrProjectionDrift
		}
		remainingAllowance += available
	}
	if remainingAllowance < delta {
		return UsageAccounting{}, ErrInsufficientProjectionAllowance
	}
	if delta == 0 {
		return result, nil
	}

	now := time.Now().UTC().Truncate(time.Second).Unix()
	remaining := delta
	allocations := make([]Allocation, 0, len(members))
	for _, member := range members {
		if remaining == 0 {
			break
		}
		available := member.allowanceBytes - member.accountedBytes
		if available == 0 {
			continue
		}
		take := available
		if take > remaining {
			take = remaining
		}

		bucketResult, err := tx.ExecContext(ctx, `
UPDATE credit_buckets
SET consumed_bytes = consumed_bytes + ?, updated_at = ?
WHERE id = ? AND proxy_user_id = ?
  AND consumed_bytes = ?
  AND consumed_bytes + ? <= original_bytes`,
			take, now, member.bucketID, userID, member.currentConsumedBytes, take)
		if err != nil {
			return UsageAccounting{}, fmt.Errorf("charge projected credit bucket: %w", err)
		}
		changed, err := bucketResult.RowsAffected()
		if err != nil {
			return UsageAccounting{}, fmt.Errorf("read projected credit charge result: %w", err)
		}
		if changed != 1 {
			return UsageAccounting{}, ErrProjectionDrift
		}

		memberResult, err := tx.ExecContext(ctx, `
UPDATE quota_projection_members
SET accounted_bytes = accounted_bytes + ?
WHERE proxy_user_id = ? AND generation = ? AND position = ? AND bucket_id = ?
  AND accounted_bytes = ?
  AND accounted_bytes + ? <= allowance_bytes`,
			take, userID, generation, member.position, member.bucketID, member.accountedBytes, take)
		if err != nil {
			return UsageAccounting{}, fmt.Errorf("advance projection member accounting: %w", err)
		}
		changed, err = memberResult.RowsAffected()
		if err != nil {
			return UsageAccounting{}, fmt.Errorf("read projection member accounting result: %w", err)
		}
		if changed != 1 {
			return UsageAccounting{}, ErrProjectionDrift
		}

		allocations = append(allocations, Allocation{BucketID: member.bucketID, Bytes: take})
		remaining -= take
	}
	if remaining != 0 {
		return UsageAccounting{}, ErrInsufficientProjectionAllowance
	}

	baselineResult, err := tx.ExecContext(ctx, `
UPDATE quota_reconciliations
SET telemt_baseline_used_bytes = ?, updated_at = ?
WHERE proxy_user_id = ? AND generation = ? AND phase = ?
  AND telemt_reset_epoch_secs = ? AND telemt_baseline_used_bytes = ?`,
		int64(observation.UsedBytes), now, userID, generation, string(PhaseActive), resetEpoch, baselineUsed)
	if err != nil {
		return UsageAccounting{}, fmt.Errorf("advance Telemt usage baseline: %w", err)
	}
	changed, err := baselineResult.RowsAffected()
	if err != nil {
		return UsageAccounting{}, fmt.Errorf("read Telemt baseline update result: %w", err)
	}
	if changed != 1 {
		return UsageAccounting{}, ErrProjectionDrift
	}

	if err := tx.Commit(); err != nil {
		return UsageAccounting{}, fmt.Errorf("commit quota usage accounting: %w", err)
	}
	result.Allocations = allocations
	return result, nil
}

type accountingMember struct {
	position             int
	bucketID             int64
	allowanceBytes       int64
	accountedBytes       int64
	originalBytes        int64
	currentConsumedBytes int64
}

func loadAccountingMembers(ctx context.Context, tx *sql.Tx, userID, generation int64) ([]accountingMember, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT m.position, m.bucket_id, m.allowance_bytes, m.accounted_bytes,
       b.original_bytes, b.consumed_bytes
FROM quota_projection_members AS m
JOIN credit_buckets AS b
  ON b.id = m.bucket_id AND b.proxy_user_id = m.proxy_user_id
WHERE m.proxy_user_id = ? AND m.generation = ?
ORDER BY m.position`, userID, generation)
	if err != nil {
		return nil, fmt.Errorf("load projection members for accounting: %w", err)
	}
	defer rows.Close()

	members := make([]accountingMember, 0)
	seenBuckets := make(map[int64]struct{})
	for rows.Next() {
		var member accountingMember
		if err := rows.Scan(
			&member.position,
			&member.bucketID,
			&member.allowanceBytes,
			&member.accountedBytes,
			&member.originalBytes,
			&member.currentConsumedBytes,
		); err != nil {
			return nil, fmt.Errorf("scan projection member for accounting: %w", err)
		}
		if member.position != len(members) || member.bucketID <= 0 || member.originalBytes <= 0 || member.allowanceBytes <= 0 || member.allowanceBytes > member.originalBytes || member.accountedBytes < 0 || member.accountedBytes > member.allowanceBytes {
			return nil, ErrProjectionDrift
		}
		if _, exists := seenBuckets[member.bucketID]; exists {
			return nil, ErrProjectionDrift
		}
		seenBuckets[member.bucketID] = struct{}{}

		consumedAtProjection := member.originalBytes - member.allowanceBytes
		if consumedAtProjection > math.MaxInt64-member.accountedBytes {
			return nil, ErrProjectionDrift
		}
		expectedCurrent := consumedAtProjection + member.accountedBytes
		if member.currentConsumedBytes != expectedCurrent {
			return nil, ErrProjectionDrift
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projection members for accounting: %w", err)
	}
	return members, nil
}
