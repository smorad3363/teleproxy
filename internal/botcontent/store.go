package botcontent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("bot content not found")
	ErrInvalid  = errors.New("bot content is invalid")
)

type Slot string

const (
	SlotWelcome    Slot = "welcome"
	SlotForcedJoin Slot = "forced_join"
	SlotReferral   Slot = "referral"
	SlotProxy      Slot = "proxy"
	SlotExpired    Slot = "expired"
	SlotNoCredit   Slot = "no_credit"
	SlotSupport    Slot = "support"
)

const MaxTextRunes = 4096

type Entry struct {
	Slot      Slot      `json:"slot"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func Set(ctx context.Context, db *sql.DB, slot Slot, text string, now time.Time) (Entry, error) {
	if db == nil {
		return Entry{}, fmt.Errorf("bot content database is required")
	}
	if !validSlot(slot) || !validText(text) {
		return Entry{}, ErrInvalid
	}
	if now.IsZero() {
		return Entry{}, fmt.Errorf("bot content time is required")
	}
	now = now.UTC().Truncate(time.Second)
	_, err := db.ExecContext(ctx, `
INSERT INTO bot_content(slot, text, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(slot) DO UPDATE SET
    text = excluded.text,
    updated_at = excluded.updated_at`,
		string(slot), text, now.Unix(), now.Unix(),
	)
	if err != nil {
		return Entry{}, fmt.Errorf("set bot content: %w", err)
	}
	return Get(ctx, db, slot)
}

func Get(ctx context.Context, db *sql.DB, slot Slot) (Entry, error) {
	if db == nil {
		return Entry{}, fmt.Errorf("bot content database is required")
	}
	if !validSlot(slot) {
		return Entry{}, ErrInvalid
	}
	entry, err := scanEntry(db.QueryRowContext(ctx, `
SELECT slot, text, created_at, updated_at
FROM bot_content
WHERE slot = ?`, string(slot)))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	return entry, err
}

func List(ctx context.Context, db *sql.DB) ([]Entry, error) {
	if db == nil {
		return nil, fmt.Errorf("bot content database is required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT slot, text, created_at, updated_at
FROM bot_content
ORDER BY slot ASC`)
	if err != nil {
		return nil, fmt.Errorf("list bot content: %w", err)
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bot content: %w", err)
	}
	return entries, nil
}

func Clear(ctx context.Context, db *sql.DB, slot Slot) error {
	if db == nil {
		return fmt.Errorf("bot content database is required")
	}
	if !validSlot(slot) {
		return ErrInvalid
	}
	result, err := db.ExecContext(ctx, "DELETE FROM bot_content WHERE slot = ?", string(slot))
	if err != nil {
		return fmt.Errorf("clear bot content: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read cleared bot content row count: %w", err)
	}
	if count != 1 {
		return ErrNotFound
	}
	return nil
}

func validSlot(slot Slot) bool {
	switch slot {
	case SlotWelcome, SlotForcedJoin, SlotReferral, SlotProxy, SlotExpired, SlotNoCredit, SlotSupport:
		return true
	default:
		return false
	}
}

func validText(text string) bool {
	if !utf8.ValidString(text) {
		return false
	}
	count := utf8.RuneCountInString(text)
	return count >= 1 && count <= MaxTextRunes
}

type rowScanner interface {
	Scan(...any) error
}

func scanEntry(row rowScanner) (Entry, error) {
	var entry Entry
	var slot string
	var createdAt, updatedAt int64
	if err := row.Scan(&slot, &entry.Text, &createdAt, &updatedAt); err != nil {
		return Entry{}, err
	}
	entry.Slot = Slot(slot)
	if !validSlot(entry.Slot) || !validText(entry.Text) || createdAt <= 0 || updatedAt < createdAt {
		return Entry{}, fmt.Errorf("stored bot content is invalid")
	}
	entry.CreatedAt = time.Unix(createdAt, 0).UTC()
	entry.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return entry, nil
}
