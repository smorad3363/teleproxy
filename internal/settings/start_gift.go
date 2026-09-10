package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	startGiftKey          = "start_gift_bytes"
	DefaultStartGiftBytes = int64(100_000_000)
)

func StartGiftBytes(ctx context.Context, db *sql.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("settings database is required")
	}
	return readStartGiftBytes(ctx, db)
}

func StartGiftBytesTx(ctx context.Context, tx *sql.Tx) (int64, error) {
	if tx == nil {
		return 0, fmt.Errorf("settings transaction is required")
	}
	return readStartGiftBytes(ctx, tx)
}

func SetStartGiftBytes(ctx context.Context, db *sql.DB, value int64) error {
	if db == nil {
		return fmt.Errorf("settings database is required")
	}
	if value <= 0 {
		return fmt.Errorf("start gift bytes must be greater than zero")
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO settings(key, value, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		startGiftKey, strconv.FormatInt(value, 10), time.Now().UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("set start gift bytes: %w", err)
	}
	return nil
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readStartGiftBytes(ctx context.Context, queryer rowQueryer) (int64, error) {
	var raw string
	err := queryer.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", startGiftKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultStartGiftBytes, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read start gift setting: %w", err)
	}
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("start gift setting is invalid")
	}
	return value, nil
}
