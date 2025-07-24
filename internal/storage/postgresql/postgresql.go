package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Storage struct {
	db *pgx.Conn
}

func New(storageConnectionString string) (*Storage, error) {
	const op = "storage.postgresql.New"

	conn, err := pgx.Connect(context.Background(), storageConnectionString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	_, err = conn.Exec(context.Background(),
	`
	CREATE TABLE IF NOT EXISTS "subscriptions"(
		"id" SERIAL PRIMARY KEY,
		"service_name" TEXT NOT NULL,
		"price" NUMERIC(10, 2) NOT NULL,
		"user_id" UUID NOT NULL,
		"start_date" DATE NOT NULL,
		"end_date" DATE);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: conn}, nil
}
