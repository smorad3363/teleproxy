package sponsor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound = errors.New("sponsor profile not found")
	ErrConflict = errors.New("sponsor profile already exists")
)

type Profile struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	ChannelRef string     `json:"channel_ref"`
	AdTag      string     `json:"ad_tag"`
	Enabled    bool       `json:"enabled"`
	Weight     int64      `json:"weight"`
	StartsAt   *time.Time `json:"starts_at,omitempty"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateProfile struct {
	Name       string
	ChannelRef string
	AdTag      string
	Enabled    bool
	Weight     int64
	StartsAt   *time.Time
	EndsAt     *time.Time
	Notes      string
}

func Create(ctx context.Context, db *sql.DB, input CreateProfile, now time.Time) (Profile, error) {
	if db == nil {
		return Profile{}, fmt.Errorf("sponsor database is required")
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
INSERT INTO sponsor_profiles(
    name, channel_ref, ad_tag, enabled, weight, starts_at, ends_at, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.Name, normalized.ChannelRef, normalized.AdTag, boolInt(normalized.Enabled), normalized.Weight,
		startsAt, endsAt, normalized.Notes, now.Unix(), now.Unix(),
	)
	if isUniqueConstraint(err) {
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, fmt.Errorf("create sponsor profile: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Profile{}, fmt.Errorf("read sponsor profile id: %w", err)
	}
	return Get(ctx, db, id)
}

func Get(ctx context.Context, db *sql.DB, id int64) (Profile, error) {
	if db == nil {
		return Profile{}, fmt.Errorf("sponsor database is required")
	}
	if id <= 0 {
		return Profile{}, ErrNotFound
	}
	profile, err := getByID(ctx, db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return profile, err
}

func List(ctx context.Context, db *sql.DB) ([]Profile, error) {
	if db == nil {
		return nil, fmt.Errorf("sponsor database is required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, name, channel_ref, ad_tag, enabled, weight, starts_at, ends_at, notes, created_at, updated_at
FROM sponsor_profiles
ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list sponsor profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]Profile, 0)
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sponsor profiles: %w", err)
	}
	return profiles, nil
}

func ValidateProfileInput(input CreateProfile) error {
	_, err := normalizeProfileInput(input)
	return err
}

func normalizeProfileInput(input CreateProfile) (CreateProfile, error) {
	name := strings.TrimSpace(input.Name)
	if !utf8.ValidString(name) || len(name) < 1 || len(name) > 128 {
		return CreateProfile{}, fmt.Errorf("sponsor name must contain between 1 and 128 UTF-8 bytes")
	}
	channelRef, err := normalizeChannelRef(input.ChannelRef)
	if err != nil {
		return CreateProfile{}, err
	}
	adTag, err := normalizeAdTag(input.AdTag)
	if err != nil {
		return CreateProfile{}, err
	}
	if input.Weight <= 0 {
		return CreateProfile{}, fmt.Errorf("sponsor weight must be greater than zero")
	}
	notes := input.Notes
	if !utf8.ValidString(notes) || len(notes) > 4096 {
		return CreateProfile{}, fmt.Errorf("sponsor notes must contain at most 4096 UTF-8 bytes")
	}

	startsAt, err := normalizeOptionalTime(input.StartsAt, "start")
	if err != nil {
		return CreateProfile{}, err
	}
	endsAt, err := normalizeOptionalTime(input.EndsAt, "end")
	if err != nil {
		return CreateProfile{}, err
	}
	if startsAt != nil && endsAt != nil && !endsAt.After(*startsAt) {
		return CreateProfile{}, fmt.Errorf("sponsor end time must be after start time")
	}

	input.Name = name
	input.ChannelRef = channelRef
	input.AdTag = adTag
	input.StartsAt = startsAt
	input.EndsAt = endsAt
	input.Notes = notes
	return input, nil
}

func normalizeChannelRef(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 1 || len(value) > 512 {
		return "", fmt.Errorf("sponsor channel reference is invalid")
	}
	if strings.HasPrefix(value, "@") {
		username := value[1:]
		if len(username) < 1 || len(username) > 64 {
			return "", fmt.Errorf("sponsor channel reference is invalid")
		}
		for _, ch := range username {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
				continue
			}
			return "", fmt.Errorf("sponsor channel reference is invalid")
		}
		return "@" + strings.ToLower(username), nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Fragment != "" || parsed.Port() != "" {
		return "", fmt.Errorf("sponsor channel reference is invalid")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "t.me" && host != "telegram.me" {
		return "", fmt.Errorf("sponsor channel reference must use a Telegram host")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return "", fmt.Errorf("sponsor channel reference is invalid")
	}
	parsed.Scheme = "https"
	parsed.Host = host
	return parsed.String(), nil
}

func normalizeAdTag(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 32 {
		return "", fmt.Errorf("sponsor ad tag must be exactly 32 hexadecimal characters")
	}
	for _, ch := range value {
		if ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' {
			continue
		}
		return "", fmt.Errorf("sponsor ad tag must be exactly 32 hexadecimal characters")
	}
	return value, nil
}

func normalizeOptionalTime(value *time.Time, label string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	if value.IsZero() {
		return nil, fmt.Errorf("sponsor %s time is invalid", label)
	}
	normalized := value.UTC().Truncate(time.Second)
	return &normalized, nil
}

func getByID(ctx context.Context, db *sql.DB, id int64) (Profile, error) {
	row := db.QueryRowContext(ctx, `
SELECT id, name, channel_ref, ad_tag, enabled, weight, starts_at, ends_at, notes, created_at, updated_at
FROM sponsor_profiles
WHERE id = ?`, id)
	return scanProfile(row)
}

type scanner interface {
	Scan(...any) error
}

func scanProfile(row scanner) (Profile, error) {
	var profile Profile
	var enabled int
	var startsAt, endsAt sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(
		&profile.ID, &profile.Name, &profile.ChannelRef, &profile.AdTag, &enabled, &profile.Weight,
		&startsAt, &endsAt, &profile.Notes, &createdAt, &updatedAt,
	); err != nil {
		return Profile{}, fmt.Errorf("read sponsor profile: %w", err)
	}
	if enabled != 0 && enabled != 1 {
		return Profile{}, fmt.Errorf("sponsor profile enabled state is invalid")
	}
	if startsAt.Valid {
		value := time.Unix(startsAt.Int64, 0).UTC()
		profile.StartsAt = &value
	}
	if endsAt.Valid {
		value := time.Unix(endsAt.Int64, 0).UTC()
		profile.EndsAt = &value
	}
	profile.Enabled = enabled == 1
	normalized, err := normalizeProfileInput(CreateProfile{
		Name: profile.Name, ChannelRef: profile.ChannelRef, AdTag: profile.AdTag, Enabled: profile.Enabled,
		Weight: profile.Weight, StartsAt: profile.StartsAt, EndsAt: profile.EndsAt, Notes: profile.Notes,
	})
	if err != nil || normalized.Name != profile.Name || normalized.ChannelRef != profile.ChannelRef || normalized.AdTag != profile.AdTag {
		return Profile{}, fmt.Errorf("stored sponsor profile is invalid")
	}
	profile.CreatedAt = time.Unix(createdAt, 0).UTC()
	profile.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return profile, nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique || sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
