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
	_, err = conn.Exec(context.Background(),
		`
	CREATE TABLE IF NOT EXISTS subscriptions(
		id SERIAL PRIMARY KEY,
		service_name TEXT NOT NULL,
		price NUMERIC(10, 2) NOT NULL,
		user_id UUID NOT NULL,
		start_date DATE NOT NULL,
		end_date DATE);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	_, err = conn.Exec(context.Background(), `
		CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id 
		ON subscriptions (user_id);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: conn}, nil
}

func (s *Storage) CreateSubscriptionEntry(ctx context.Context, entry subs.CreaterSubscriptionEntry) (int, error) {

	const op = "storage.postgresql.CreateSubscriptionEntry"
	var result int
	err := s.db.QueryRow(ctx, `
		INSERT INTO subscriptions (
			service_name,
			price,
			user_id,
			start_date,
			end_date
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		entry.ServiceName,
		entry.Price,
		entry.UserID,
		entry.StartDate,
		entry.EndDate).Scan(&result)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil

}

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
