package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
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

	if err := initializeSchema(conn); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: conn}, nil
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

func (s *Storage) CreateSubscriptionEntry(ctx context.Context, entry subs.CreaterSubscriptionEntry) (int, error) {
	const op = "storage.postgresql.CreateSubscriptionEntry"

	_, err := s.db.Exec(ctx, `
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
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM subscriptions").Scan(&res)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return res, nil
}

// исходя из задания не понял, могу ли я удалять пользователей
func (s *Storage) RemoveSubscriptionEntryByUserID(ctx context.Context, entry subs.FilterRemoverSubscriptionEntry) (int64, error) {

	const op = "storage.postgresql.DeleteSubscriptionEntryByUserID"

	commandTag, err := s.db.Exec(ctx, `
		DELETE FROM subscriptions WHERE user_id = $1`, entry.UserID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()

	return result, nil
}

// исходя из задания не понял, могу ли я удалять пользователей
func (s *Storage) RemoveSubscriptionEntryByServiceName(ctx context.Context, entry subs.FilterRemoverSubscriptionEntry) (int64, error) {

	const op = "storage.postgresql.DeleteSubscriptionEntryByServiceName"

	commandTag, err := s.db.Exec(ctx, `
		DELETE FROM subscriptions WHERE service_name = $1 and user_id = $2`, entry.ServiceName, entry.UserID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()

	return result, nil
}

func (s *Storage) ReadSubscriptionEntryByUserID(ctx context.Context, entry subs.FilterReaderSubscriptionEntry) ([]*subs.FilterReaderSubscriptionEntry, error) {

	const op = "storage.postgresql.ReadSubscriptionEntryByUserID"

	rows, err := s.db.Query(ctx, `
		SELECT service_name, price, user_id, start_date, end_date 
		FROM subscriptions WHERE user_id = $1`, entry.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var result []*subs.FilterReaderSubscriptionEntry
	for rows.Next() {
		var item subs.FilterReaderSubscriptionEntry
		err := rows.Scan(&item.ServiceName, &item.Price, &item.UserID, &item.StartDate, &item.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	return result, nil
}

func (s *Storage) UpdateSubscriptionEntryPriceByServiceName(ctx context.Context, entry subs.FilterUpdaterSubscriptionEntry) (int64, error) {
	const op = "storage.postgresql.UpdateSubscriptionEntryByServiceNamePrice"

	commandTag, err := s.db.Exec(ctx, `
		UPDATE subscriptions SET price = $1 WHERE user_id = $2 AND service_name = $3`,
		entry.Price, entry.UserID, entry.ServiceName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()
	return result, nil
}

func (s *Storage) UpdateSubscriptionEntryDateByServiceName(ctx context.Context, entry subs.FilterUpdaterSubscriptionEntry) (int64, error) {
	const op = "storage.postgresql.UpdateSubscriptionEntryByService"

	commandTag, err := s.db.Exec(ctx, `
		UPDATE subscriptions
		SET start_date = $1, end_date = $2
		WHERE user_id = $3 AND service_name = $4`,
		entry.StartDate, entry.EndDate, entry.UserID, entry.ServiceName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()
	return result, nil
}

func (s *Storage) ListSubscriptionEntrys(ctx context.Context) ([]*subs.ListSubscriptionEntrys, error) {
	const op = "storage.postgresql.ListSubscriptionEntrys"

	rows, err := s.db.Query(ctx, `
		SELECT service_name, price, user_id, start_date, end_date
		FROM subscriptions`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var result []*subs.ListSubscriptionEntrys
	for rows.Next() {
		var item subs.ListSubscriptionEntrys
		err := rows.Scan(&item.ServiceName, &item.Price, &item.UserID, &item.StartDate, &item.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	return result, nil

}

func (s *Storage) CountSumSubscriptionEntrys(ctx context.Context, entry subs.CounterSumSubscriptionEntrys) (float64, error) {
	const op = "storage.postgresql.CountSumSubscriptionEntrys"

	var res *float64

	err := s.db.QueryRow(ctx, `
		SELECT SUM(price)
		FROM subscriptions
		WHERE user_id = $1
			AND service_name = $2
			AND start_date <= $3
			AND (end_date IS NULL OR end_date >= $4)`,
		entry.UserID, entry.ServiceName, entry.EndDate, entry.StartDate).Scan(&res)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if res == nil {
		return 0, nil
	}

	return *res, nil
}

