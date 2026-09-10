package sponsor

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func Update(ctx context.Context, db *sql.DB, id int64, input CreateProfile, now time.Time) (Profile, error) {
	if db == nil {
		return Profile{}, fmt.Errorf("sponsor database is required")
	}
	if id <= 0 {
		return Profile{}, ErrNotFound
	}
	if now.IsZero() {
		return Profile{}, fmt.Errorf("sponsor time is required")
	}
	normalized, err := normalizeProfileInput(input)
	if err != nil {
		return Profile{}, err
	}
	now = now.UTC().Truncate(time.Second)

	var startsAt, endsAt any
	if normalized.StartsAt != nil {
		startsAt = normalized.StartsAt.Unix()
	}
	if normalized.EndsAt != nil {
		endsAt = normalized.EndsAt.Unix()
	}
	result, err := db.ExecContext(ctx, `
UPDATE sponsor_profiles
SET name = ?, channel_ref = ?, ad_tag = ?, enabled = ?, weight = ?, starts_at = ?, ends_at = ?, notes = ?, updated_at = ?
WHERE id = ?`,
		normalized.Name, normalized.ChannelRef, normalized.AdTag, boolInt(normalized.Enabled), normalized.Weight,
		startsAt, endsAt, normalized.Notes, now.Unix(), id,
	)
	if isUniqueConstraint(err) {
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, fmt.Errorf("update sponsor profile: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Profile{}, fmt.Errorf("read updated sponsor profile row count: %w", err)
	}
	if count != 1 {
		return Profile{}, ErrNotFound
	}
	return Get(ctx, db, id)
}

func Delete(ctx context.Context, db *sql.DB, id int64) error {
	if db == nil {
		return fmt.Errorf("sponsor database is required")
	}
	if id <= 0 {
		return ErrNotFound
	}
	result, err := db.ExecContext(ctx, "DELETE FROM sponsor_profiles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete sponsor profile: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted sponsor profile row count: %w", err)
	}
	if count != 1 {
		return ErrNotFound
	}
	return nil
}
