package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	busyTimeoutMS = 5000
)

func Open(ctx context.Context, path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("database path is required")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// SQLite PRAGMAs such as foreign_keys and busy_timeout are connection-scoped.
	// The MVP intentionally uses one DB connection so those invariants cannot be
	// bypassed by a newly-created pooled connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	closeOnError := func(err error) (*sql.DB, error) {
		_ = db.Close()
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return closeOnError(fmt.Errorf("ping sqlite database: %w", err))
	}

	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		return closeOnError(fmt.Errorf("enable sqlite WAL: %w", err))
	}
	if !strings.EqualFold(journalMode, "wal") {
		return closeOnError(fmt.Errorf("enable sqlite WAL: got journal_mode=%q", journalMode))
	}

	pragmas := []string{
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeoutMS),
	}
	for _, statement := range pragmas {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return closeOnError(fmt.Errorf("apply sqlite pragma %q: %w", statement, err))
		}
	}

	if err := Migrate(ctx, db); err != nil {
		return closeOnError(err)
	}

	return db, nil
}
