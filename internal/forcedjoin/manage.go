package forcedjoin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func List(ctx context.Context, db *sql.DB) ([]Channel, error) {
	if db == nil {
		return nil, fmt.Errorf("forced join database is required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, chat_ref, display_name, join_url, enabled, required, position, custom_text, created_at, updated_at
FROM forced_join_channels
ORDER BY position ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list forced join channels: %w", err)
	}
	defer rows.Close()

	channels := make([]Channel, 0)
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate forced join channels: %w", err)
	}
	return channels, nil
}

func Get(ctx context.Context, db *sql.DB, id int64) (Channel, error) {
	if db == nil {
		return Channel{}, fmt.Errorf("forced join database is required")
	}
	if id <= 0 {
		return Channel{}, ErrNotFound
	}
	channel, err := getByID(ctx, db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Channel{}, ErrNotFound
	}
	return channel, err
}

func Update(ctx context.Context, db *sql.DB, id int64, input CreateChannel, now time.Time) (Channel, error) {
	if db == nil {
		return Channel{}, fmt.Errorf("forced join database is required")
	}
	if id <= 0 {
		return Channel{}, ErrNotFound
	}
	if now.IsZero() {
		return Channel{}, fmt.Errorf("forced join time is required")
	}
	normalized, err := normalizeChannelInput(input)
	if err != nil {
		return Channel{}, err
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
UPDATE forced_join_channels
SET chat_ref = ?, display_name = ?, join_url = ?, enabled = ?, required = ?, position = ?, custom_text = ?, updated_at = ?
WHERE id = ?`,
		normalized.ChatRef, normalized.DisplayName, normalized.JoinURL, boolInt(normalized.Enabled), boolInt(normalized.Required),
		normalized.Position, normalized.CustomText, now.Unix(), id,
	)
	if isUniqueConstraint(err) {
		return Channel{}, ErrConflict
	}
	if err != nil {
		return Channel{}, fmt.Errorf("update forced join channel: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Channel{}, fmt.Errorf("read updated forced join row count: %w", err)
	}
	if count != 1 {
		return Channel{}, ErrNotFound
	}
	return Get(ctx, db, id)
}

func Delete(ctx context.Context, db *sql.DB, id int64) error {
	if db == nil {
		return fmt.Errorf("forced join database is required")
	}
	if id <= 0 {
		return ErrNotFound
	}
	result, err := db.ExecContext(ctx, "DELETE FROM forced_join_channels WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete forced join channel: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted forced join row count: %w", err)
	}
	if count != 1 {
		return ErrNotFound
	}
	return nil
}
