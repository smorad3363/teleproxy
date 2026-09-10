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
	referralRewardBytesKey      = "referral_reward_bytes"
	referralRewardExpiryDaysKey = "referral_reward_expiry_days"
	DefaultReferralRewardBytes  = int64(2_000_000_000)
	DefaultReferralRewardDays   = int64(14)
)

type ReferralRewardSettings struct {
	Bytes      int64 `json:"bytes"`
	ExpiryDays int64 `json:"expiry_days"`
}

func ReferralReward(ctx context.Context, db *sql.DB) (ReferralRewardSettings, error) {
	if db == nil {
		return ReferralRewardSettings{}, fmt.Errorf("settings database is required")
	}
	return readReferralReward(ctx, db)
}

func ReferralRewardTx(ctx context.Context, tx *sql.Tx) (ReferralRewardSettings, error) {
	if tx == nil {
		return ReferralRewardSettings{}, fmt.Errorf("settings transaction is required")
	}
	return readReferralReward(ctx, tx)
}

func SetReferralReward(ctx context.Context, db *sql.DB, bytes, expiryDays int64) error {
	if db == nil {
		return fmt.Errorf("settings database is required")
	}
	if err := validateReferralReward(bytes, expiryDays); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin referral reward settings transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Unix()
	if err := upsertReferralSetting(ctx, tx, referralRewardBytesKey, bytes, now); err != nil {
		return err
	}
	if err := upsertReferralSetting(ctx, tx, referralRewardExpiryDaysKey, expiryDays, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit referral reward settings: %w", err)
	}
	return nil
}

func readReferralReward(ctx context.Context, queryer rowQueryer) (ReferralRewardSettings, error) {
	bytes, err := readPositiveSetting(ctx, queryer, referralRewardBytesKey, DefaultReferralRewardBytes)
	if err != nil {
		return ReferralRewardSettings{}, fmt.Errorf("read referral reward bytes: %w", err)
	}
	expiryDays, err := readPositiveSetting(ctx, queryer, referralRewardExpiryDaysKey, DefaultReferralRewardDays)
	if err != nil {
		return ReferralRewardSettings{}, fmt.Errorf("read referral reward expiry days: %w", err)
	}
	return ReferralRewardSettings{Bytes: bytes, ExpiryDays: expiryDays}, nil
}

func readPositiveSetting(ctx context.Context, queryer rowQueryer, key string, fallback int64) (int64, error) {
	var raw string
	err := queryer.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return fallback, nil
	}
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("stored value is invalid")
	}
	return value, nil
}

func validateReferralReward(bytes, expiryDays int64) error {
	if bytes <= 0 {
		return fmt.Errorf("referral reward bytes must be greater than zero")
	}
	if expiryDays <= 0 {
		return fmt.Errorf("referral reward expiry days must be greater than zero")
	}
	return nil
}

func upsertReferralSetting(ctx context.Context, tx *sql.Tx, key string, value, updatedAt int64) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO settings(key, value, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, strconv.FormatInt(value, 10), updatedAt,
	)
	if err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}
