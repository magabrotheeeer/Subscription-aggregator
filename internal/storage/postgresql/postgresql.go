package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type Storage struct {
	Db *pgx.Conn
}

func New(storageConnectionString string) (*Storage, error) {
	const op = "storage.postgresql.New"

	conn, err := pgx.Connect(context.Background(), storageConnectionString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := initializeSchema(conn); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{Db: conn}, nil
}

func initializeSchema(conn *pgx.Conn) error {

	_, err := conn.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS schema_info(
            key TEXT PRIMARY KEY,
            value TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `)
	if err != nil {
		return fmt.Errorf("failed to create schema_info: %w", err)
	}

	var exists bool
	err = conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM schema_info WHERE key = 'initialized')").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check initialization: %w", err)
	}

	if !exists {
		tx, err := conn.Begin(context.Background())
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		defer tx.Rollback(context.Background())

		_, err = tx.Exec(context.Background(), `
            CREATE TABLE IF NOT EXISTS subscriptions(
                id SERIAL PRIMARY KEY,
                service_name TEXT NOT NULL,
                price NUMERIC(10, 2) NOT NULL,
                user_id UUID NOT NULL,
                start_date DATE NOT NULL,
                end_date DATE
            );
        `)
		if err != nil {
			return fmt.Errorf("failed to create subscriptions table: %w", err)
		}

		_, err = tx.Exec(context.Background(), `
            CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id 
            ON subscriptions (user_id);
        `)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}

		_, err = tx.Exec(context.Background(),
			"INSERT INTO schema_info (key, value) VALUES ('initialized', 'true')")
		if err != nil {
			return fmt.Errorf("failed to mark as initialized: %w", err)
		}

		if err = tx.Commit(context.Background()); err != nil {
			return fmt.Errorf("failed to commit initialization: %w", err)
		}
	}

	return nil
}

func (s *Storage) CreateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry) (int, error) {
	const op = "storage.postgresql.CreateSubscriptionEntry"

	_, err := s.Db.Exec(ctx, `
        INSERT INTO subscriptions (
            service_name,
            price,
            user_id,
            start_date,
            end_date
        ) VALUES ($1, $2, $3, $4, $5)`,
		entry.ServiceName,
		entry.Price,
		entry.UserID,
		entry.StartDate,
		entry.EndDate)

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	var res int
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM subscriptions").Scan(&res)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return res, nil
}

func (s *Storage) RemoveSubscriptionEntry(ctx context.Context, id int) (int64, error) {

	const op = "storage.postgresql.DeleteSubscriptionEntryByUserID"

	commandTag, err := s.Db.Exec(ctx, `
		DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()

	return result, nil
}

func (s *Storage) ReadSubscriptionEntry(ctx context.Context, id int) ([]*subs.SubscriptionEntry, error) {

	const op = "storage.postgresql.ReadSubscriptionEntryByUserID"

	rows, err := s.Db.Query(ctx, `
		SELECT service_name, price, user_id, start_date, end_date 
		FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var result []*subs.SubscriptionEntry
	for rows.Next() {
		var item subs.SubscriptionEntry
		err := rows.Scan(&item.ServiceName, &item.Price, &item.UserID, &item.StartDate, &item.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	return result, nil
}

func (s *Storage) UpdateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry, id int) (int64, error) {
	const op = "storage.postgresql.UpdateSubscriptionEntryByServiceNamePrice"

	commandTag, err := s.Db.Exec(ctx, `
		UPDATE subscriptions SET service_name = $1, start_date = $2, end_date = $3, price = $4, user_id = $5
			WHERE id = $6`,
		entry.ServiceName, entry.StartDate, entry.EndDate, entry.Price, entry.UserID, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()
	return result, nil
}

func (s *Storage) ListSubscriptionEntrys(ctx context.Context) ([]*subs.SubscriptionEntry, error) {
	const op = "storage.postgresql.ListSubscriptionEntrys"

	rows, err := s.Db.Query(ctx, `
		SELECT service_name, price, user_id, start_date, end_date
		FROM subscriptions`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var result []*subs.SubscriptionEntry
	for rows.Next() {
		var item subs.SubscriptionEntry
		err := rows.Scan(&item.ServiceName, &item.Price, &item.UserID, &item.StartDate, &item.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	return result, nil

}

func (s *Storage) CountSumSubscriptionEntrys(ctx context.Context, entry subs.SubscriptionEntry, id int) (float64, error) {
	const op = "storage.postgresql.CountSumSubscriptionEntrys"

	var res *float64

	err := s.Db.QueryRow(ctx, `
		SELECT SUM(price)
		FROM subscriptions
		WHERE user_id = $1
			AND service_name = $2
			AND start_date <= $3
			AND (end_date IS NULL OR end_date >= $4)
			AND id = $5`,
		entry.UserID, entry.ServiceName, entry.EndDate, entry.StartDate, id).Scan(&res)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if res == nil {
		return 0, nil
	}

	return *res, nil
}
