package proxyuser

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func Get(ctx context.Context, db *sql.DB, username string) (User, error) {
	if db == nil {
		return User{}, fmt.Errorf("proxy user database is required")
	}
	username = strings.TrimSpace(username)
	if err := ValidateUsername(username); err != nil {
		return User{}, err
	}
	return get(ctx, db, username)
}
