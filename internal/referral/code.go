package referral

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

func EnsureCode(ctx context.Context, db *sql.DB, telegramUserID int64, now time.Time) (Code, error) {
	if db == nil {
		return Code{}, fmt.Errorf("referral database is required")
	}
	if telegramUserID <= 0 {
		return Code{}, fmt.Errorf("Telegram user ID must be greater than zero")
	}
	if now.IsZero() {
		return Code{}, fmt.Errorf("referral code time is required")
	}
	if existing, err := GetCode(ctx, db, telegramUserID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrCodeNotFound) {
		return Code{}, err
	}
	if err := ensureTelegramUserExists(ctx, db, telegramUserID); err != nil {
		return Code{}, err
	}

	now = now.UTC().Truncate(time.Second)
	for attempt := 0; attempt < codeCreateAttempts; attempt++ {
		value, err := generateCode()
		if err != nil {
			return Code{}, err
		}
		_, err = db.ExecContext(ctx, `
INSERT INTO referral_codes(telegram_user_id, code, created_at)
VALUES (?, ?, ?)`, telegramUserID, value, now.Unix())
		if err == nil {
			return Code{TelegramUserID: telegramUserID, Value: value, CreatedAt: now}, nil
		}
		if !isUniqueConstraint(err) {
			return Code{}, fmt.Errorf("create referral code: %w", err)
		}
		if existing, getErr := GetCode(ctx, db, telegramUserID); getErr == nil {
			return existing, nil
		} else if !errors.Is(getErr, ErrCodeNotFound) {
			return Code{}, getErr
		}
	}
	return Code{}, fmt.Errorf("create referral code: unique code attempts exhausted")
}

func GetCode(ctx context.Context, db *sql.DB, telegramUserID int64) (Code, error) {
	if db == nil {
		return Code{}, fmt.Errorf("referral database is required")
	}
	if telegramUserID <= 0 {
		return Code{}, fmt.Errorf("Telegram user ID must be greater than zero")
	}
	var code Code
	var createdAt int64
	err := db.QueryRowContext(ctx, `
SELECT telegram_user_id, code, created_at
FROM referral_codes
WHERE telegram_user_id = ?`, telegramUserID).Scan(&code.TelegramUserID, &code.Value, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Code{}, ErrCodeNotFound
	}
	if err != nil {
		return Code{}, fmt.Errorf("read referral code: %w", err)
	}
	if !validCode(code.Value) {
		return Code{}, fmt.Errorf("stored referral code is invalid")
	}
	code.CreatedAt = time.Unix(createdAt, 0).UTC()
	return code, nil
}

func getCodeByValue(ctx context.Context, db *sql.DB, value string) (Code, error) {
	var code Code
	var createdAt int64
	err := db.QueryRowContext(ctx, `
SELECT telegram_user_id, code, created_at
FROM referral_codes
WHERE code = ?`, value).Scan(&code.TelegramUserID, &code.Value, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Code{}, ErrCodeNotFound
	}
	if err != nil {
		return Code{}, fmt.Errorf("resolve referral code: %w", err)
	}
	if !validCode(code.Value) {
		return Code{}, fmt.Errorf("stored referral code is invalid")
	}
	code.CreatedAt = time.Unix(createdAt, 0).UTC()
	return code, nil
}

func ensureTelegramUserExists(ctx context.Context, db *sql.DB, telegramUserID int64) error {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM telegram_users WHERE id = ?", telegramUserID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTelegramUserNotFound
	}
	if err != nil {
		return fmt.Errorf("verify referral Telegram user: %w", err)
	}
	return nil
}

func generateCode() (string, error) {
	buffer := make([]byte, codeRandomBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate referral code: %w", err)
	}
	value := base64.RawURLEncoding.EncodeToString(buffer)
	if !validCode(value) {
		return "", fmt.Errorf("generated referral code is invalid")
	}
	return value, nil
}

func validCode(value string) bool {
	if len(value) < 16 || len(value) > 64 {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}
