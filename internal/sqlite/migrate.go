package sqlite

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	sqlitemigrate "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(sqlitePath string, migrationsDir string) error {
	migrationDB, err := Open(sqlitePath)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer migrationDB.Close()

	driver, err := sqlitemigrate.WithInstance(migrationDB, &sqlitemigrate.Config{})
	if err != nil {
		return fmt.Errorf("create sqlite migration driver: %w", err)
	}

	migrations, err := migrate.NewWithDatabaseInstance("file://"+migrationsDir, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	defer migrations.Close()

	if err := migrations.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
