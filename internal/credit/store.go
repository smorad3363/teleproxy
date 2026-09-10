package credit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrUserNotFound       = errors.New("proxy user not found")
	ErrBucketNotFound     = errors.New("credit bucket not found")
	ErrInsufficientCredit = errors.New("insufficient credit")
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusExhausted Status = "exhausted"
	StatusExpired   Status = "expired"
	StatusRevoked   Status = "revoked"
)

type Grant struct {
	OriginalBytes int64
	StartsAt      time.Time
	ExpiresAt     *time.Time
	RewardType    string
	Source        string
}

type Bucket struct {
	ID            int64      `json:"id"`
	ProxyUserID   int64      `json:"proxy_user_id"`
	OriginalBytes int64      `json:"original_bytes"`
	ConsumedBytes int64      `json:"consumed_bytes"`
	StartsAt      time.Time  `json:"starts_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	RewardType    string     `json:"reward_type"`
	Source        string     `json:"source"`
	Status        Status     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Allocation struct {
	BucketID int64 `json:"bucket_id"`
	Bytes    int64 `json:"bytes"`
}

type Consumption struct {
	RequestedBytes int64        `json:"requested_bytes"`
	ConsumedBytes  int64        `json:"consumed_bytes"`
	Allocations    []Allocation `json:"allocations"`
}

func GrantBucket(ctx context.Context, db *sql.DB, username string, grant Grant) (Bucket, error) {
	if db == nil {
		return Bucket{}, fmt.Errorf("credit database is required")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return Bucket{}, fmt.Errorf("proxy username is required")
	}
	grant.RewardType = strings.TrimSpace(grant.RewardType)
	grant.Source = strings.TrimSpace(grant.Source)
	if grant.OriginalBytes <= 0 {
		return Bucket{}, fmt.Errorf("original bytes must be greater than zero")
	}
	if grant.StartsAt.IsZero() {
		return Bucket{}, fmt.Errorf("start time is required")
	}
	startsAt := grant.StartsAt.UTC().Truncate(time.Second)
	var expiresAt *time.Time
	if grant.ExpiresAt != nil {
		value := grant.ExpiresAt.UTC().Truncate(time.Second)
		if !value.After(startsAt) {
			return Bucket{}, fmt.Errorf("expiry must be after start time")
		}
		expiresAt = &value
	}
	if err := validateLabel("reward type", grant.RewardType, 64); err != nil {
		return Bucket{}, err
	}
	if err := validateLabel("source", grant.Source, 256); err != nil {
		return Bucket{}, err
	}

	userID, err := proxyUserID(ctx, db, username)
	if err != nil {
		return Bucket{}, err
	}
	now := time.Now().UTC().Unix()
	var expires any
	if expiresAt != nil {
		expires = expiresAt.Unix()
	}
	result, err := db.ExecContext(ctx, `
INSERT INTO credit_buckets(
    proxy_user_id, original_bytes, consumed_bytes, starts_at, expires_at,
    reward_type, source, status, created_at, updated_at
) VALUES (?, ?, 0, ?, ?, ?, ?, 'active', ?, ?)`,
		userID, grant.OriginalBytes, startsAt.Unix(), expires,
		grant.RewardType, grant.Source, now, now,
	)
	if err != nil {
		return Bucket{}, fmt.Errorf("grant credit bucket: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Bucket{}, fmt.Errorf("read credit bucket id: %w", err)
	}
	return Bucket{
		ID:            id,
		ProxyUserID:   userID,
		OriginalBytes: grant.OriginalBytes,
		StartsAt:      startsAt,
		ExpiresAt:     expiresAt,
		RewardType:    grant.RewardType,
		Source:        grant.Source,
		Status:        effectiveStatus("active", 0, grant.OriginalBytes, startsAt, expiresAt, time.Now().UTC()),
		CreatedAt:     time.Unix(now, 0).UTC(),
		UpdatedAt:     time.Unix(now, 0).UTC(),
	}, nil
}

func List(ctx context.Context, db *sql.DB, username string, now time.Time) ([]Bucket, error) {
	if db == nil {
		return nil, fmt.Errorf("credit database is required")
	}
	userID, err := proxyUserID(ctx, db, strings.TrimSpace(username))
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, proxy_user_id, original_bytes, consumed_bytes, starts_at, expires_at,
       reward_type, source, status, created_at, updated_at
FROM credit_buckets
WHERE proxy_user_id = ?
ORDER BY starts_at, id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list credit buckets: %w", err)
	}
	defer rows.Close()

	buckets := make([]Bucket, 0)
	for rows.Next() {
		bucket, err := scanBucket(rows, now)
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit buckets: %w", err)
	}
	return buckets, nil
}

func Balance(ctx context.Context, db *sql.DB, username string, now time.Time) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("credit database is required")
	}
	userID, err := proxyUserID(ctx, db, strings.TrimSpace(username))
	if err != nil {
		return 0, err
	}
	var balance sql.NullInt64
	err = db.QueryRowContext(ctx, `
SELECT SUM(original_bytes - consumed_bytes)
FROM credit_buckets
WHERE proxy_user_id = ?
  AND status = 'active'
  AND starts_at <= ?
  AND (expires_at IS NULL OR expires_at > ?)
  AND consumed_bytes < original_bytes`, userID, now.UTC().Unix(), now.UTC().Unix()).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("calculate credit balance: %w", err)
	}
	if !balance.Valid {
		return 0, nil
	}
	return balance.Int64, nil
}

func Revoke(ctx context.Context, db *sql.DB, username string, bucketID int64) (Bucket, error) {
	if db == nil {
		return Bucket{}, fmt.Errorf("credit database is required")
	}
	if bucketID <= 0 {
		return Bucket{}, ErrBucketNotFound
	}
	userID, err := proxyUserID(ctx, db, strings.TrimSpace(username))
	if err != nil {
		return Bucket{}, err
	}
	now := time.Now().UTC()
	result, err := db.ExecContext(ctx, `
UPDATE credit_buckets
SET status = 'revoked', updated_at = ?
WHERE id = ? AND proxy_user_id = ?`, now.Unix(), bucketID, userID)
	if err != nil {
		return Bucket{}, fmt.Errorf("revoke credit bucket: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Bucket{}, fmt.Errorf("read credit revoke result: %w", err)
	}
	if changed == 0 {
		return Bucket{}, ErrBucketNotFound
	}
	return bucketByID(ctx, db, userID, bucketID, now)
}

func Consume(ctx context.Context, db *sql.DB, username string, bytes int64, now time.Time) (Consumption, error) {
	if db == nil {
		return Consumption{}, fmt.Errorf("credit database is required")
	}
	if bytes <= 0 {
		return Consumption{}, fmt.Errorf("consumed bytes must be greater than zero")
	}
	now = now.UTC().Truncate(time.Second)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Consumption{}, fmt.Errorf("begin credit consumption: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, err := proxyUserIDQuery(ctx, tx, strings.TrimSpace(username))
	if err != nil {
		return Consumption{}, err
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id, original_bytes - consumed_bytes
FROM credit_buckets
WHERE proxy_user_id = ?
  AND status = 'active'
  AND starts_at <= ?
  AND (expires_at IS NULL OR expires_at > ?)
  AND consumed_bytes < original_bytes
ORDER BY CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END,
         expires_at ASC,
         starts_at ASC,
         id ASC`, userID, now.Unix(), now.Unix())
	if err != nil {
		return Consumption{}, fmt.Errorf("select consumable credit: %w", err)
	}
	type availableBucket struct {
		id        int64
		remaining int64
	}
	available := make([]availableBucket, 0)
	var total int64
	for rows.Next() {
		var bucket availableBucket
		if err := rows.Scan(&bucket.id, &bucket.remaining); err != nil {
			_ = rows.Close()
			return Consumption{}, fmt.Errorf("scan consumable credit: %w", err)
		}
		if bucket.remaining <= 0 {
			continue
		}
		if bucket.remaining >= bytes-total {
			total = bytes
		} else {
			total += bucket.remaining
		}
		available = append(available, bucket)
		if total >= bytes {
			break
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return Consumption{}, fmt.Errorf("iterate consumable credit: %w", err)
	}
	if err := rows.Close(); err != nil {
		return Consumption{}, fmt.Errorf("close consumable credit rows: %w", err)
	}
	if total < bytes {
		return Consumption{}, ErrInsufficientCredit
	}

	remaining := bytes
	allocations := make([]Allocation, 0, len(available))
	for _, bucket := range available {
		if remaining == 0 {
			break
		}
		take := bucket.remaining
		if take > remaining {
			take = remaining
		}
		result, err := tx.ExecContext(ctx, `
UPDATE credit_buckets
SET consumed_bytes = consumed_bytes + ?, updated_at = ?
WHERE id = ?
  AND proxy_user_id = ?
  AND status = 'active'
  AND consumed_bytes + ? <= original_bytes`, take, now.Unix(), bucket.id, userID, take)
		if err != nil {
			return Consumption{}, fmt.Errorf("consume credit bucket: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return Consumption{}, fmt.Errorf("read credit consumption result: %w", err)
		}
		if changed != 1 {
			return Consumption{}, fmt.Errorf("credit bucket changed during consumption")
		}
		allocations = append(allocations, Allocation{BucketID: bucket.id, Bytes: take})
		remaining -= take
	}
	if remaining != 0 {
		return Consumption{}, fmt.Errorf("credit allocation did not satisfy requested bytes")
	}
	if err := tx.Commit(); err != nil {
		return Consumption{}, fmt.Errorf("commit credit consumption: %w", err)
	}
	return Consumption{RequestedBytes: bytes, ConsumedBytes: bytes, Allocations: allocations}, nil
}

func validateLabel(name, value string, max int) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(value) > max {
		return fmt.Errorf("%s must be at most %d bytes", name, max)
	}
	return nil
}

func proxyUserID(ctx context.Context, db *sql.DB, username string) (int64, error) {
	return proxyUserIDQuery(ctx, db, username)
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func proxyUserIDQuery(ctx context.Context, queryer rowQuerier, username string) (int64, error) {
	if username == "" {
		return 0, ErrUserNotFound
	}
	var id int64
	err := queryer.QueryRowContext(ctx, "SELECT id FROM proxy_users WHERE username = ?", username).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("find proxy user for credit: %w", err)
	}
	return id, nil
}

func bucketByID(ctx context.Context, db *sql.DB, userID, bucketID int64, now time.Time) (Bucket, error) {
	row := db.QueryRowContext(ctx, `
SELECT id, proxy_user_id, original_bytes, consumed_bytes, starts_at, expires_at,
       reward_type, source, status, created_at, updated_at
FROM credit_buckets
WHERE proxy_user_id = ? AND id = ?`, userID, bucketID)
	bucket, err := scanBucket(row, now)
	if errors.Is(err, sql.ErrNoRows) {
		return Bucket{}, ErrBucketNotFound
	}
	return bucket, err
}

type scanner interface {
	Scan(...any) error
}

func scanBucket(row scanner, now time.Time) (Bucket, error) {
	var bucket Bucket
	var startsAt, createdAt, updatedAt int64
	var expiresAt sql.NullInt64
	var storedStatus string
	if err := row.Scan(
		&bucket.ID,
		&bucket.ProxyUserID,
		&bucket.OriginalBytes,
		&bucket.ConsumedBytes,
		&startsAt,
		&expiresAt,
		&bucket.RewardType,
		&bucket.Source,
		&storedStatus,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Bucket{}, err
	}
	bucket.StartsAt = time.Unix(startsAt, 0).UTC()
	if expiresAt.Valid {
		value := time.Unix(expiresAt.Int64, 0).UTC()
		bucket.ExpiresAt = &value
	}
	bucket.CreatedAt = time.Unix(createdAt, 0).UTC()
	bucket.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	bucket.Status = effectiveStatus(storedStatus, bucket.ConsumedBytes, bucket.OriginalBytes, bucket.StartsAt, bucket.ExpiresAt, now.UTC())
	return bucket, nil
}

func effectiveStatus(stored string, consumed, original int64, startsAt time.Time, expiresAt *time.Time, now time.Time) Status {
	if stored == "revoked" {
		return StatusRevoked
	}
	if consumed >= original {
		return StatusExhausted
	}
	if startsAt.After(now) {
		return StatusPending
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return StatusExpired
	}
	return StatusActive
}
