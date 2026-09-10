package referral

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	DefaultHistoryLimit = 50
	MaxHistoryLimit     = 100
)

type HistoryQuery struct {
	BeforeID int64
	Limit    int
}

type HistoryEntry struct {
	ID                int64      `json:"id"`
	InviterUserID     int64      `json:"inviter_user_id"`
	InviterTelegramID int64      `json:"inviter_telegram_id"`
	InviteeUserID     int64      `json:"invitee_user_id"`
	InviteeTelegramID int64      `json:"invitee_telegram_id"`
	Status            Status     `json:"status"`
	RejectionReason   *string    `json:"rejection_reason,omitempty"`
	EligibleAt        *time.Time `json:"eligible_at,omitempty"`
	FinalizedAt       *time.Time `json:"finalized_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type HistoryPage struct {
	Items        []HistoryEntry `json:"items"`
	NextBeforeID *int64         `json:"next_before_id,omitempty"`
}

func History(ctx context.Context, db *sql.DB, query HistoryQuery) (HistoryPage, error) {
	if db == nil {
		return HistoryPage{}, fmt.Errorf("referral database is required")
	}
	if query.BeforeID < 0 {
		return HistoryPage{}, fmt.Errorf("history before ID must not be negative")
	}
	if query.Limit == 0 {
		query.Limit = DefaultHistoryLimit
	}
	if query.Limit < 1 || query.Limit > MaxHistoryLimit {
		return HistoryPage{}, fmt.Errorf("history limit must be between 1 and %d", MaxHistoryLimit)
	}

	statement := `
SELECT
    ra.id,
    ra.inviter_user_id,
    inviter.telegram_id,
    ra.invitee_user_id,
    invitee.telegram_id,
    ra.status,
    ra.rejection_reason,
    ra.eligible_at,
    ra.finalized_at,
    ra.created_at,
    ra.updated_at
FROM referral_attributions AS ra
JOIN telegram_users AS inviter ON inviter.id = ra.inviter_user_id
JOIN telegram_users AS invitee ON invitee.id = ra.invitee_user_id`
	args := make([]any, 0, 2)
	if query.BeforeID > 0 {
		statement += "\nWHERE ra.id < ?"
		args = append(args, query.BeforeID)
	}
	statement += "\nORDER BY ra.id DESC\nLIMIT ?"
	args = append(args, query.Limit+1)

	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return HistoryPage{}, fmt.Errorf("query referral history: %w", err)
	}
	defer rows.Close()

	items := make([]HistoryEntry, 0, query.Limit+1)
	for rows.Next() {
		entry, err := scanHistoryEntry(rows)
		if err != nil {
			return HistoryPage{}, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return HistoryPage{}, fmt.Errorf("iterate referral history: %w", err)
	}

	page := HistoryPage{Items: items}
	if len(page.Items) > query.Limit {
		page.Items = page.Items[:query.Limit]
		next := page.Items[len(page.Items)-1].ID
		page.NextBeforeID = &next
	}
	return page, nil
}

func scanHistoryEntry(row scanner) (HistoryEntry, error) {
	var entry HistoryEntry
	var status string
	var rejectionReason sql.NullString
	var eligibleAt, finalizedAt sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(
		&entry.ID,
		&entry.InviterUserID,
		&entry.InviterTelegramID,
		&entry.InviteeUserID,
		&entry.InviteeTelegramID,
		&status,
		&rejectionReason,
		&eligibleAt,
		&finalizedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return HistoryEntry{}, fmt.Errorf("read referral history entry: %w", err)
	}
	entry.Status = Status(status)
	if entry.Status != StatusPending && entry.Status != StatusRewarded && entry.Status != StatusRejected {
		return HistoryEntry{}, fmt.Errorf("stored referral attribution status is invalid")
	}
	if rejectionReason.Valid {
		value := rejectionReason.String
		entry.RejectionReason = &value
	}
	if eligibleAt.Valid {
		value := time.Unix(eligibleAt.Int64, 0).UTC()
		entry.EligibleAt = &value
	}
	if finalizedAt.Valid {
		value := time.Unix(finalizedAt.Int64, 0).UTC()
		entry.FinalizedAt = &value
	}
	entry.CreatedAt = time.Unix(createdAt, 0).UTC()
	entry.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return entry, nil
}
