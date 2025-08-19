package migrations

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Run(db *sql.DB, path string) error {
	const op = "migrations.Run"
	driver, err := pgxv5.WithInstance(db, &pgxv5.Config{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+path,
		"pgx_v5",
		driver,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
