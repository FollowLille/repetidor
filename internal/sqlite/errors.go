package sqlite

import (
	"errors"

	sqlite3 "modernc.org/sqlite"
)

const (
	sqliteConstraintPrimaryKey = 1555
	sqliteConstraintUnique     = 2067
)

func isUniqueConstraintError(err error) bool {
	var sqliteErr *sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return sqliteErr.Code() == sqliteConstraintUnique || sqliteErr.Code() == sqliteConstraintPrimaryKey
}
