package proxynode

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func Update(ctx context.Context, db *sql.DB, id int64, input CreateNode, now time.Time) (Node, error) {
	if db == nil {
		return Node{}, fmt.Errorf("proxy node database is required")
	}
	if id <= 0 {
		return Node{}, ErrNotFound
	}
	if now.IsZero() {
		return Node{}, fmt.Errorf("proxy node time is required")
	}
	normalized, err := normalizeInput(input)
	if err != nil {
		return Node{}, err
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
UPDATE proxy_nodes
SET node_type = ?, name = ?, region = ?, host = ?, public_host = ?, mtproto_port = ?, internal_api_endpoint = ?, enabled = ?, updated_at = ?
WHERE id = ?`,
		string(normalized.Type), normalized.Name, normalized.Region, normalized.Host, normalized.PublicHost,
		normalized.MTProtoPort, normalized.InternalAPIEndpoint, boolInt(normalized.Enabled), now.Unix(), id,
	)
	if isUniqueConstraint(err) {
		return Node{}, ErrConflict
	}
	if err != nil {
		return Node{}, fmt.Errorf("update proxy node: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Node{}, fmt.Errorf("read updated proxy node row count: %w", err)
	}
	if count != 1 {
		return Node{}, ErrNotFound
	}
	return Get(ctx, db, id)
}

func Delete(ctx context.Context, db *sql.DB, id int64) error {
	if db == nil {
		return fmt.Errorf("proxy node database is required")
	}
	if id <= 0 {
		return ErrNotFound
	}
	result, err := db.ExecContext(ctx, "DELETE FROM proxy_nodes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete proxy node: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted proxy node row count: %w", err)
	}
	if count != 1 {
		return ErrNotFound
	}
	return nil
}
