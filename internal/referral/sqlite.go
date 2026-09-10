package referral

import (
	"errors"

	"github.com/mattn/go-sqlite3"
)

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
