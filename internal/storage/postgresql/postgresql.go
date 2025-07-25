package postgresql

import (
	"context"
	"fmt"
	"time"

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

func (s *Storage) CreateSubscriptionEntry(ctx context.Context, serviceName string, price int,
	 userID string, startDate time.Time, endDate time.Time) (int, error) {

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
		serviceName,
		price,
		userID,
		startDate,
		endDate).Scan(&result)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil

}

func (s *Storage) RemoveSubscriptionEntryByUserID(ctx context.Context, userID string) (int64, error) {

	const op = "storage.postgresql.DeleteSubscriptionEntryByUserID"

	commandTag, err := s.db.Exec(ctx, `
		DELETE FROM subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()

	return result, nil
}

func (s *Storage) RemoveSubscriptionEntryByServiceName(ctx context.Context, serviceName, userID string) (int64, error) {

	const op = "storage.postgresql.DeleteSubscriptionEntryByServiceName"

	commandTag, err := s.db.Exec(ctx, `
		DELETE FROM subscriptions WHERE service_name = $1 and user_id = $2`, serviceName, userID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()

	return result, nil
}

/*func (s *Storage) ReadSubscriptionEntryByUserID(ctx context.Context, userID string) ([]struct, error) {
	
	const op = "storage.postgresql.ReadSubscriptionEntryByUserID"

	rows, err := s.db.Query(ctx, `
		SELECT FROM subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	subscriptions := []struct{
		ServiceName string
		Price int
		UserID string
		StartDate time.Time
		EndDate time.Time
	}{}

	for rows.Next() {
		var item struct{
			ServiceName string
			Price int
			UserID string
			StartDate time.Time
			EndDate time.Time
		}
		err := rows.Scan(&item.ServiceName, &item.Price, &item.UserID, &item.StartDate, &item.EndDate)
		if err != nil {
			return 
		}
	}
}*/

// TO-DO
