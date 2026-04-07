package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func Migrate(db *sql.DB, dir string) error {
	if db == nil {
		return fmt.Errorf("migrate: db is required")
	}

	if dir == "" {
		return fmt.Errorf("migrate: migrations directory is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("migrate: read directory %q: %w", dir, err)
	}

	files := make([]string, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}

		files = append(files, filepath.Join(dir, name))
	}

	sort.Strings(files)

	for _, filePath := range files {
		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("migrate: read file %q: %w", filePath, err)
		}

		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("migrate: execute file %q: %w", filePath, err)
		}
	}
	return nil
}
