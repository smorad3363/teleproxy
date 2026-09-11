package auditlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxFieldBytes    = 256
	maxSnapshotBytes = 64 << 10
	maxListLimit     = 100
)

var (
	ErrInvalid  = errors.New("audit log input is invalid")
	ErrNotFound = errors.New("audit log entry not found")
)

type Entry struct {
	ID             int64     `json:"id"`
	Actor          string    `json:"actor"`
	Action         string    `json:"action"`
	Target         string    `json:"target"`
	BeforeSnapshot *string   `json:"before,omitempty"`
	AfterSnapshot  *string   `json:"after,omitempty"`
	RequestID      string    `json:"request_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type AppendInput struct {
	Actor          string
	Action         string
	Target         string
	BeforeSnapshot *string
	AfterSnapshot  *string
	RequestID      string
}

func Append(ctx context.Context, db *sql.DB, input AppendInput, now time.Time) (Entry, error) {
	if db == nil {
		return Entry{}, fmt.Errorf("audit log database is required")
	}
	if now.IsZero() {
		return Entry{}, fmt.Errorf("audit log time is required")
	}
	if err := ValidateAppendInput(input); err != nil {
		return Entry{}, err
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
INSERT INTO audit_log(actor, action, target, before_snapshot, after_snapshot, request_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.Actor, input.Action, input.Target, nullableString(input.BeforeSnapshot), nullableString(input.AfterSnapshot), input.RequestID, now.Unix(),
	)
	if err != nil {
		return Entry{}, fmt.Errorf("append audit log entry: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Entry{}, fmt.Errorf("read audit log entry id: %w", err)
	}
	return Get(ctx, db, id)
}

func Get(ctx context.Context, db *sql.DB, id int64) (Entry, error) {
	if db == nil {
		return Entry{}, fmt.Errorf("audit log database is required")
	}
	if id <= 0 {
		return Entry{}, ErrNotFound
	}
	row := db.QueryRowContext(ctx, `
SELECT id, actor, action, target, before_snapshot, after_snapshot, request_id, created_at
FROM audit_log
WHERE id = ?`, id)
	entry, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	return entry, err
}

func List(ctx context.Context, db *sql.DB, beforeID int64, limit int) ([]Entry, error) {
	if db == nil {
		return nil, fmt.Errorf("audit log database is required")
	}
	if beforeID < 0 || limit < 1 || limit > maxListLimit {
		return nil, ErrInvalid
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, actor, action, target, before_snapshot, after_snapshot, request_id, created_at
FROM audit_log
WHERE (? = 0 OR id < ?)
ORDER BY id DESC
LIMIT ?`, beforeID, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit log entries: %w", err)
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
		return nil, fmt.Errorf("iterate audit log entries: %w", err)
	}
	return entries, nil
}

func ValidateAppendInput(input AppendInput) error {
	for _, value := range []string{input.Actor, input.Action, input.Target, input.RequestID} {
		if !validRequiredField(value) {
			return ErrInvalid
		}
	}
	if !validSnapshot(input.BeforeSnapshot) || !validSnapshot(input.AfterSnapshot) {
		return ErrInvalid
	}
	return nil
}

func validRequiredField(value string) bool {
	return utf8.ValidString(value) && value == strings.TrimSpace(value) && len(value) >= 1 && len(value) <= maxFieldBytes
}

func validSnapshot(value *string) bool {
	return value == nil || utf8.ValidString(*value) && len(*value) <= maxSnapshotBytes
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(row scanner) (Entry, error) {
	var entry Entry
	var before, after sql.NullString
	var createdAt int64
	if err := row.Scan(&entry.ID, &entry.Actor, &entry.Action, &entry.Target, &before, &after, &entry.RequestID, &createdAt); err != nil {
		return Entry{}, err
	}
	if entry.ID <= 0 || !validRequiredField(entry.Actor) || !validRequiredField(entry.Action) || !validRequiredField(entry.Target) || !validRequiredField(entry.RequestID) || createdAt <= 0 {
		return Entry{}, fmt.Errorf("invalid persisted audit log entry")
	}
	if before.Valid {
		entry.BeforeSnapshot = &before.String
	}
	if after.Valid {
		entry.AfterSnapshot = &after.String
	}
	if !validSnapshot(entry.BeforeSnapshot) || !validSnapshot(entry.AfterSnapshot) {
		return Entry{}, fmt.Errorf("invalid persisted audit log snapshot")
	}
	entry.CreatedAt = time.Unix(createdAt, 0).UTC()
	return entry, nil
}
