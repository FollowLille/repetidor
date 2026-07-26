package sqlite

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return sql.Open("sqlite", path+separator+"_pragma=foreign_keys(1)")
}
