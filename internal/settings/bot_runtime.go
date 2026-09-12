package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type BotRuntimeSettings struct {
	Username    string `json:"username"`
	AdminChatID int64  `json:"admin_chat_id"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   int64  `json:"updated_at"`
}

func BotRuntime(ctx context.Context, db *sql.DB) (BotRuntimeSettings, bool, error) {
	if db == nil {
		return BotRuntimeSettings{}, false, fmt.Errorf("settings database is required")
	}
	var value BotRuntimeSettings
	var enabled int
	err := db.QueryRowContext(ctx, `
SELECT username, admin_chat_id, enabled, updated_at
FROM bot_runtime_settings
WHERE id = 1`).Scan(&value.Username, &value.AdminChatID, &enabled, &value.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BotRuntimeSettings{}, false, nil
	}
	if err != nil {
		return BotRuntimeSettings{}, false, fmt.Errorf("read bot runtime settings: %w", err)
	}
	value.Enabled = enabled == 1
	if err := ValidateBotRuntime(value); err != nil {
		return BotRuntimeSettings{}, false, fmt.Errorf("stored bot runtime settings are invalid: %w", err)
	}
	return value, true, nil
}

func SetBotRuntime(ctx context.Context, db *sql.DB, value BotRuntimeSettings, now time.Time) (BotRuntimeSettings, error) {
	if db == nil {
		return BotRuntimeSettings{}, fmt.Errorf("settings database is required")
	}
	value.Username = NormalizeBotUsername(value.Username)
	if err := ValidateBotRuntime(value); err != nil {
		return BotRuntimeSettings{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}
	value.UpdatedAt = now.UTC().Unix()
	enabled := 0
	if value.Enabled {
		enabled = 1
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO bot_runtime_settings(id, username, admin_chat_id, enabled, created_at, updated_at)
VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    username = excluded.username,
    admin_chat_id = excluded.admin_chat_id,
    enabled = excluded.enabled,
    updated_at = excluded.updated_at`,
		value.Username, value.AdminChatID, enabled, value.UpdatedAt, value.UpdatedAt,
	)
	if err != nil {
		return BotRuntimeSettings{}, fmt.Errorf("set bot runtime settings: %w", err)
	}
	return value, nil
}

func ValidateBotRuntime(value BotRuntimeSettings) error {
	username := NormalizeBotUsername(value.Username)
	if username == "" || len(username) > 64 {
		return fmt.Errorf("Telegram Bot username must contain between 1 and 64 characters")
	}
	for _, ch := range username {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return fmt.Errorf("Telegram Bot username contains unsupported characters")
	}
	if value.AdminChatID <= 0 {
		return fmt.Errorf("administrator Telegram chat ID must be positive")
	}
	return nil
}

func NormalizeBotUsername(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "@")
}
