// Package repository реализует хранилище данных на основе PostgreSQL
// для управления подписками и пользователями. Предоставляет методы
// создания, чтения, обновления, удаления и агрегирования записей,
// а также работу с пользователями.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	// Регистрация драйвера pgx для использования с database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Storage инкапсулирует соединение с базой данных PostgreSQL
// и реализует методы работы с подписками и пользователями.
type Storage struct {
	DB *sql.DB
}

// New создаёт подключение к PostgreSQL и инициализирует необходимые таблицы и индексы.
func New(storageConnectionString string) (*Storage, error) {
	const op = "storage.New"

	db, err := sql.Open("pgx", storageConnectionString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err = db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		DB: db,
	}, nil
}

// CheckDatabaseReady проверяет готовность базы данных.
func CheckDatabaseReady(storage *Storage) error {
	var exists bool
	err := storage.DB.QueryRow(`SELECT EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_name = 'subscriptions'
    )`).Scan(&exists)
	if err != nil || !exists {
		return fmt.Errorf("required table subscriptions missing or query error: %w", err)
	}
	return nil
}
