package forcedjoin

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Channel struct {
	ID          int64
	ChatRef     string
	DisplayName string
	JoinURL     string
	Enabled     bool
	Required    bool
	Position    int
	CustomText  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateChannel struct {
	ChatRef     string
	DisplayName string
	JoinURL     string
	Enabled     bool
	Required    bool
	Position    int
	CustomText  string
}

func Create(ctx context.Context, db *sql.DB, input CreateChannel, now time.Time) (Channel, error) {
	if db == nil {
		return Channel{}, fmt.Errorf("forced join database is required")
	}
	if now.IsZero() {
		return Channel{}, fmt.Errorf("forced join time is required")
	}
	chatRef, err := normalizeChatRef(input.ChatRef)
	if err != nil {
		return Channel{}, err
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if !utf8.ValidString(displayName) || len(displayName) < 1 || len(displayName) > 128 {
		return Channel{}, fmt.Errorf("forced join display name must contain between 1 and 128 UTF-8 bytes")
	}
	joinURL, err := normalizeJoinURL(input.JoinURL)
	if err != nil {
		return Channel{}, err
	}
	if input.Position < 0 || input.Position > 1000000 {
		return Channel{}, fmt.Errorf("forced join position is out of range")
	}
	customText := input.CustomText
	if !utf8.ValidString(customText) || len(customText) > 1024 {
		return Channel{}, fmt.Errorf("forced join custom text must contain at most 1024 UTF-8 bytes")
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
INSERT INTO forced_join_channels(
    chat_ref, display_name, join_url, enabled, required, position, custom_text, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chatRef, displayName, joinURL, boolInt(input.Enabled), boolInt(input.Required), input.Position,
		customText, now.Unix(), now.Unix(),
	)
	if err != nil {
		return Channel{}, fmt.Errorf("create forced join channel: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Channel{}, fmt.Errorf("read forced join channel id: %w", err)
	}
	return getByID(ctx, db, id)
}

func ListRequired(ctx context.Context, db *sql.DB) ([]Channel, error) {
	if db == nil {
		return nil, fmt.Errorf("forced join database is required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, chat_ref, display_name, join_url, enabled, required, position, custom_text, created_at, updated_at
FROM forced_join_channels
WHERE enabled = 1 AND required = 1
ORDER BY position ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list required forced join channels: %w", err)
	}
	defer rows.Close()
	var channels []Channel
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

func getByID(ctx context.Context, db *sql.DB, id int64) (Channel, error) {
	row := db.QueryRowContext(ctx, `
SELECT id, chat_ref, display_name, join_url, enabled, required, position, custom_text, created_at, updated_at
FROM forced_join_channels WHERE id = ?`, id)
	return scanChannel(row)
}

type scanner interface {
	Scan(...any) error
}

func scanChannel(row scanner) (Channel, error) {
	var channel Channel
	var enabled, required int
	var createdAt, updatedAt int64
	if err := row.Scan(
		&channel.ID, &channel.ChatRef, &channel.DisplayName, &channel.JoinURL,
		&enabled, &required, &channel.Position, &channel.CustomText, &createdAt, &updatedAt,
	); err != nil {
		return Channel{}, fmt.Errorf("read forced join channel: %w", err)
	}
	if enabled != 0 && enabled != 1 || required != 0 && required != 1 {
		return Channel{}, fmt.Errorf("forced join channel boolean state is invalid")
	}
	chatRef, err := normalizeChatRef(channel.ChatRef)
	if err != nil || chatRef != channel.ChatRef {
		return Channel{}, fmt.Errorf("forced join channel reference is invalid")
	}
	joinURL, err := normalizeJoinURL(channel.JoinURL)
	if err != nil || joinURL != channel.JoinURL {
		return Channel{}, fmt.Errorf("forced join channel URL is invalid")
	}
	if !utf8.ValidString(channel.DisplayName) || len(channel.DisplayName) < 1 || len(channel.DisplayName) > 128 {
		return Channel{}, fmt.Errorf("forced join channel display name is invalid")
	}
	if !utf8.ValidString(channel.CustomText) || len(channel.CustomText) > 1024 {
		return Channel{}, fmt.Errorf("forced join channel custom text is invalid")
	}
	channel.Enabled = enabled == 1
	channel.Required = required == 1
	channel.CreatedAt = time.Unix(createdAt, 0).UTC()
	channel.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return channel, nil
}

func normalizeChatRef(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 1 || len(value) > 128 {
		return "", fmt.Errorf("forced join chat reference is invalid")
	}
	if strings.HasPrefix(value, "@") {
		username := value[1:]
		if len(username) < 1 || len(username) > 64 {
			return "", fmt.Errorf("forced join chat reference is invalid")
		}
		for _, ch := range username {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
				continue
			}
			return "", fmt.Errorf("forced join chat reference is invalid")
		}
		return "@" + username, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id == 0 {
		return "", fmt.Errorf("forced join chat reference is invalid")
	}
	return strconv.FormatInt(id, 10), nil
}

func normalizeJoinURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 1 || len(value) > 512 {
		return "", fmt.Errorf("forced join URL is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Fragment != "" || parsed.Port() != "" {
		return "", fmt.Errorf("forced join URL is invalid")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "t.me" && host != "telegram.me" {
		return "", fmt.Errorf("forced join URL must use a Telegram host")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return "", fmt.Errorf("forced join URL is invalid")
	}
	return parsed.String(), nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
