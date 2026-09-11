package useradmin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 100
)

type ListQuery struct {
	BeforeID      int64
	Limit         int
	TelegramID    int64
	ProxyUsername string
}

type Entry struct {
	TelegramUserID    int64                 `json:"telegram_user_id"`
	TelegramID        int64                 `json:"telegram_id"`
	ProxyUserID       int64                 `json:"proxy_user_id"`
	ProxyUsername     string                `json:"proxy_username"`
	DesiredEnabled    bool                  `json:"enabled"`
	SyncState         proxyuser.SyncState   `json:"sync_state"`
	ProvisioningPhase *proxyprovision.Phase `json:"provisioning_phase"`
	LastErrorCode     string                `json:"last_error_code,omitempty"`
	AvailableBytes    int64                 `json:"available_bytes"`
	NearestExpiry     *time.Time            `json:"nearest_expiry,omitempty"`
	ReferralCount     int64                 `json:"referral_count"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

type Page struct {
	Items        []Entry `json:"items"`
	NextBeforeID *int64  `json:"next_before_id,omitempty"`
}

func List(ctx context.Context, db *sql.DB, query ListQuery, now time.Time) (Page, error) {
	if db == nil {
		return Page{}, fmt.Errorf("user inventory database is required")
	}
	if now.IsZero() {
		return Page{}, fmt.Errorf("user inventory timestamp is required")
	}
	if query.BeforeID < 0 {
		return Page{}, fmt.Errorf("user inventory before ID must not be negative")
	}
	if query.TelegramID < 0 {
		return Page{}, fmt.Errorf("user inventory Telegram ID must not be negative")
	}
	if query.ProxyUsername != "" {
		if err := proxyuser.ValidateUsername(query.ProxyUsername); err != nil {
			return Page{}, fmt.Errorf("user inventory proxy username is invalid: %w", err)
		}
	}
	if query.Limit == 0 {
		query.Limit = DefaultListLimit
	}
	if query.Limit < 1 || query.Limit > MaxListLimit {
		return Page{}, fmt.Errorf("user inventory limit must be between 1 and %d", MaxListLimit)
	}

	nowUnix := now.UTC().Unix()
	statement := `
SELECT
    tu.id,
    tu.telegram_id,
    pu.id,
    pu.username,
    pu.desired_enabled,
    pu.sync_state,
    pp.phase,
    COALESCE(pu.last_error_code, ''),
    COALESCE((
        SELECT SUM(cb.original_bytes - cb.consumed_bytes)
        FROM credit_buckets AS cb
        WHERE cb.proxy_user_id = pu.id
          AND cb.status = 'active'
          AND cb.starts_at <= ?
          AND (cb.expires_at IS NULL OR cb.expires_at > ?)
          AND cb.consumed_bytes < cb.original_bytes
    ), 0),
    (
        SELECT MIN(cb.expires_at)
        FROM credit_buckets AS cb
        WHERE cb.proxy_user_id = pu.id
          AND cb.status = 'active'
          AND cb.starts_at <= ?
          AND cb.expires_at IS NOT NULL
          AND cb.expires_at > ?
          AND cb.consumed_bytes < cb.original_bytes
    ),
    (
        SELECT COUNT(*)
        FROM referral_attributions AS ra
        WHERE ra.inviter_user_id = tu.id
    ),
    tu.created_at,
    tu.updated_at
FROM telegram_users AS tu
JOIN proxy_users AS pu ON pu.id = tu.proxy_user_id
LEFT JOIN proxy_user_provisioning AS pp ON pp.proxy_user_id = pu.id`
	args := []any{nowUnix, nowUnix, nowUnix, nowUnix}
	conditions := make([]string, 0, 3)
	if query.BeforeID > 0 {
		conditions = append(conditions, "tu.id < ?")
		args = append(args, query.BeforeID)
	}
	if query.TelegramID > 0 {
		conditions = append(conditions, "tu.telegram_id = ?")
		args = append(args, query.TelegramID)
	}
	if query.ProxyUsername != "" {
		conditions = append(conditions, "pu.username = ?")
		args = append(args, query.ProxyUsername)
	}
	if len(conditions) > 0 {
		statement += "\nWHERE " + strings.Join(conditions, " AND ")
	}
	statement += "\nORDER BY tu.id DESC\nLIMIT ?"
	args = append(args, query.Limit+1)

	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return Page{}, fmt.Errorf("query user inventory: %w", err)
	}
	defer rows.Close()

	items := make([]Entry, 0, query.Limit+1)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return Page{}, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate user inventory: %w", err)
	}

	page := Page{Items: items}
	if len(page.Items) > query.Limit {
		page.Items = page.Items[:query.Limit]
		next := page.Items[len(page.Items)-1].TelegramUserID
		page.NextBeforeID = &next
	}
	return page, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(row scanner) (Entry, error) {
	var entry Entry
	var desiredEnabled int64
	var syncState string
	var provisioningPhase sql.NullString
	var nearestExpiry sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(
		&entry.TelegramUserID,
		&entry.TelegramID,
		&entry.ProxyUserID,
		&entry.ProxyUsername,
		&desiredEnabled,
		&syncState,
		&provisioningPhase,
		&entry.LastErrorCode,
		&entry.AvailableBytes,
		&nearestExpiry,
		&entry.ReferralCount,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Entry{}, fmt.Errorf("read user inventory entry: %w", err)
	}
	if desiredEnabled != 0 && desiredEnabled != 1 {
		return Entry{}, fmt.Errorf("stored proxy desired state is invalid")
	}
	entry.DesiredEnabled = desiredEnabled == 1
	entry.SyncState = proxyuser.SyncState(syncState)
	if entry.SyncState != proxyuser.SyncPending && entry.SyncState != proxyuser.SyncSynced && entry.SyncState != proxyuser.SyncError {
		return Entry{}, fmt.Errorf("stored proxy sync state is invalid")
	}
	if provisioningPhase.Valid {
		phase := proxyprovision.Phase(provisioningPhase.String)
		switch phase {
		case proxyprovision.PhasePrepared, proxyprovision.PhaseOwned, proxyprovision.PhaseCollision:
			entry.ProvisioningPhase = &phase
		default:
			return Entry{}, fmt.Errorf("stored proxy provisioning phase is invalid")
		}
	}
	if entry.LastErrorCode != "" && !safeErrorCode(entry.LastErrorCode) {
		return Entry{}, fmt.Errorf("stored proxy error code is invalid")
	}
	if nearestExpiry.Valid {
		value := time.Unix(nearestExpiry.Int64, 0).UTC()
		entry.NearestExpiry = &value
	}
	entry.CreatedAt = time.Unix(createdAt, 0).UTC()
	entry.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return entry, nil
}

func safeErrorCode(code string) bool {
	if len(code) < 1 || len(code) > 64 {
		return false
	}
	for _, r := range code {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}
