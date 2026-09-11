package admin

import (
	"context"
	"database/sql"
	"time"
)

type InventoryEntry struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ListInventory(ctx context.Context, db *sql.DB) ([]InventoryEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, username, role, enabled, created_at, updated_at
		FROM admins
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]InventoryEntry, 0)
	for rows.Next() {
		var entry InventoryEntry
		var enabled int
		var createdAt int64
		var updatedAt int64
		if err := rows.Scan(&entry.ID, &entry.Username, &entry.Role, &enabled, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		entry.Enabled = enabled != 0
		entry.CreatedAt = time.Unix(createdAt, 0).UTC()
		entry.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
