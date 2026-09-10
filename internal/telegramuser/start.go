package telegramuser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/settings"
)

var (
	ErrNotFound              = errors.New("Telegram user not found")
	ErrProxyUsernameConflict = errors.New("generated proxy username already exists")
	ErrStartInvariant        = errors.New("Telegram start state is inconsistent")
)

const (
	startGiftRewardType = "start_gift"
	startGiftSource     = "telegram:/start"
)

type User struct {
	ID            int64     `json:"id"`
	TelegramID    int64     `json:"telegram_id"`
	ProxyUserID   int64     `json:"proxy_user_id"`
	ProxyUsername string    `json:"proxy_username"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StartResult struct {
	User             User  `json:"user"`
	Created          bool  `json:"created"`
	InitialGiftBytes int64 `json:"initial_gift_bytes"`
}

func ProxyUsernameForTelegramID(telegramID int64) (string, error) {
	if telegramID <= 0 {
		return "", fmt.Errorf("Telegram ID must be greater than zero")
	}
	username := "tg_" + strconv.FormatInt(telegramID, 10)
	if err := proxyuser.ValidateUsername(username); err != nil {
		return "", fmt.Errorf("generated proxy username: %w", err)
	}
	return username, nil
}

func Start(ctx context.Context, db *sql.DB, telegramID int64, now time.Time) (StartResult, error) {
	if db == nil {
		return StartResult{}, fmt.Errorf("Telegram user database is required")
	}
	if telegramID <= 0 {
		return StartResult{}, fmt.Errorf("Telegram ID must be greater than zero")
	}
	if now.IsZero() {
		return StartResult{}, fmt.Errorf("start time is required")
	}
	now = now.UTC().Truncate(time.Second)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return StartResult{}, fmt.Errorf("begin Telegram start bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	user, err := getByTelegramID(ctx, tx, telegramID)
	if err == nil {
		giftBytes, err := existingStartGift(ctx, tx, user.ProxyUserID, telegramID)
		if err != nil {
			return StartResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return StartResult{}, fmt.Errorf("commit Telegram start replay: %w", err)
		}
		return StartResult{User: user, Created: false, InitialGiftBytes: giftBytes}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return StartResult{}, err
	}

	giftBytes, err := settings.StartGiftBytesTx(ctx, tx)
	if err != nil {
		return StartResult{}, err
	}
	username, err := ProxyUsernameForTelegramID(telegramID)
	if err != nil {
		return StartResult{}, err
	}
	var occupied int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM proxy_users WHERE username = ?", username).Scan(&occupied)
	if err == nil {
		return StartResult{}, ErrProxyUsernameConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return StartResult{}, fmt.Errorf("check generated proxy username: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
INSERT INTO proxy_users(username, desired_enabled, sync_state, created_at, updated_at)
VALUES (?, 1, 'pending', ?, ?)`, username, now.Unix(), now.Unix())
	if err != nil {
		return StartResult{}, fmt.Errorf("create proxy user for Telegram user: %w", err)
	}
	proxyUserID, err := result.LastInsertId()
	if err != nil {
		return StartResult{}, fmt.Errorf("read proxy user id for Telegram user: %w", err)
	}

	result, err = tx.ExecContext(ctx, `
INSERT INTO telegram_users(telegram_id, proxy_user_id, created_at, updated_at)
VALUES (?, ?, ?, ?)`, telegramID, proxyUserID, now.Unix(), now.Unix())
	if err != nil {
		return StartResult{}, fmt.Errorf("create Telegram user: %w", err)
	}
	telegramUserID, err := result.LastInsertId()
	if err != nil {
		return StartResult{}, fmt.Errorf("read Telegram user id: %w", err)
	}

	idempotencyKey := startGiftKey(telegramID)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO credit_buckets(
    proxy_user_id, original_bytes, consumed_bytes, starts_at, expires_at,
    reward_type, source, status, created_at, updated_at, idempotency_key
) VALUES (?, ?, 0, ?, NULL, ?, ?, 'active', ?, ?, ?)`,
		proxyUserID, giftBytes, now.Unix(), startGiftRewardType, startGiftSource,
		now.Unix(), now.Unix(), idempotencyKey,
	); err != nil {
		return StartResult{}, fmt.Errorf("create initial Telegram start gift: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return StartResult{}, fmt.Errorf("commit Telegram start bootstrap: %w", err)
	}
	return StartResult{
		User: User{
			ID:            telegramUserID,
			TelegramID:    telegramID,
			ProxyUserID:   proxyUserID,
			ProxyUsername: username,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Created:          true,
		InitialGiftBytes: giftBytes,
	}, nil
}

func Get(ctx context.Context, db *sql.DB, telegramID int64) (User, error) {
	if db == nil {
		return User{}, fmt.Errorf("Telegram user database is required")
	}
	if telegramID <= 0 {
		return User{}, fmt.Errorf("Telegram ID must be greater than zero")
	}
	return getByTelegramID(ctx, db, telegramID)
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getByTelegramID(ctx context.Context, queryer rowQueryer, telegramID int64) (User, error) {
	var user User
	var createdAt, updatedAt int64
	err := queryer.QueryRowContext(ctx, `
SELECT tu.id, tu.telegram_id, tu.proxy_user_id, pu.username, tu.created_at, tu.updated_at
FROM telegram_users tu
JOIN proxy_users pu ON pu.id = tu.proxy_user_id
WHERE tu.telegram_id = ?`, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.ProxyUserID,
		&user.ProxyUsername,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("read Telegram user: %w", err)
	}
	user.CreatedAt = time.Unix(createdAt, 0).UTC()
	user.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return user, nil
}

func existingStartGift(ctx context.Context, queryer rowQueryer, proxyUserID, telegramID int64) (int64, error) {
	var count int
	var originalBytes sql.NullInt64
	err := queryer.QueryRowContext(ctx, `
SELECT COUNT(*), MAX(original_bytes)
FROM credit_buckets
WHERE proxy_user_id = ? AND idempotency_key = ?`, proxyUserID, startGiftKey(telegramID)).Scan(&count, &originalBytes)
	if err != nil {
		return 0, fmt.Errorf("verify initial Telegram start gift: %w", err)
	}
	if count != 1 || !originalBytes.Valid || originalBytes.Int64 <= 0 {
		return 0, ErrStartInvariant
	}
	return originalBytes.Int64, nil
}

func startGiftKey(telegramID int64) string {
	return "start-gift:telegram:" + strconv.FormatInt(telegramID, 10)
}
