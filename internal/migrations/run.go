// Package migrations реализует функцию для применения миграций базы данных.
//
// Run выполняет миграции SQL, находящиеся в указанной директории,
// используя подключение к базе данных и драйвер pgx/v5.
// В случае ошибок возвращает их с контекстом операции.
package migrations

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"

	// Импорт необходим для регистрации драйвера источника миграций.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Run применяет все необработанные миграции из директории path к базе данных db.
//
// Использует драйвер pgx/v5 для взаимодействия с базой PostgreSQL.
// Возвращает ошибку при сбое миграции, кроме случая, когда изменений нет.
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
