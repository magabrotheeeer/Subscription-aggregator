package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/count_sum"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
	"github.com/magabrotheeeer/subscription-aggregator/internal/user"
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

	_, err = conn.Exec(context.Background(), `
            CREATE TABLE IF NOT EXISTS subscriptions(
                id SERIAL PRIMARY KEY,
                service_name TEXT NOT NULL,
                price NUMERIC(10, 2) NOT NULL,
                username VARCHAR(255) NOT NULL,
                start_date DATE NOT NULL,
                end_date DATE);
        `)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = conn.Exec(context.Background(), `
            CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id 
            ON subscriptions (username);
        `)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = conn.Exec(context.Background(), `
			CREATE TABLE IF NOT EXISTS users(
				id SERIAL PRIMARY KEY,
				username VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);	
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	_, err = conn.Exec(context.Background(), `
            CREATE INDEX IF NOT EXISTS idx_users_username
            ON users (username);
        `)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{Db: conn}, nil
}

func (s *Storage) CreateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry) (int, error) {
	const op = "storage.postgresql.CreateSubscriptionEntry"
	var newId int
	err := s.Db.QueryRow(ctx, `
        INSERT INTO subscriptions (
            service_name,
            price,
            username,
            start_date,
            end_date
        ) VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		entry.ServiceName,
		entry.Price,
		entry.Username,
		entry.StartDate,
		entry.EndDate).Scan(&newId)

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return newId, nil
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

func (s *Storage) ReadSubscriptionEntry(ctx context.Context, id int) (*subs.SubscriptionEntry, error) {

	const op = "storage.postgresql.ReadSubscriptionEntryByUserID"

	row := s.Db.QueryRow(ctx, `
		SELECT service_name, price, username, start_date, end_date 
		FROM subscriptions WHERE id = $1`, id)
	var result subs.SubscriptionEntry
	err := row.Scan(&result.ServiceName, &result.Price, &result.Username, &result.StartDate, &result.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &result, nil
}

func (s *Storage) UpdateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry, id int) (int64, error) {
	const op = "storage.postgresql.UpdateSubscriptionEntryByServiceNamePrice"

	commandTag, err := s.Db.Exec(ctx, `
		UPDATE subscriptions
		SET service_name = $1,
			start_date = $2,
			end_date = $3,
			price = $4,
			username = $5
		WHERE id = $6`,
		entry.ServiceName, entry.StartDate, entry.EndDate, entry.Price, entry.Username, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	result := commandTag.RowsAffected()
	return result, nil
}

func (s *Storage) ListSubscriptionEntrys(ctx context.Context, username string, limit, offset int) ([]*subs.SubscriptionEntry, error) {
	const op = "storage.postgresql.ListSubscriptionEntrys"

	rows, err := s.Db.Query(ctx, `
		SELECT service_name, price, username, start_date, end_date
		FROM subscriptions LIMIT $1 OFFSET $2
		WHERE username = $3`,
		limit, offset, username)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var result []*subs.SubscriptionEntry
	for rows.Next() {
		var item subs.SubscriptionEntry
		err := rows.Scan(&item.ServiceName, &item.Price, &item.Username, &item.StartDate, &item.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	return result, nil

}

func (s *Storage) CountSumSubscriptionEntrys(ctx context.Context, entry countsum.SubscriptionFilterSum) (float64, error) {
	const op = "storage.postgresql.CountSumSubscriptionEntrys"

	var res sql.NullFloat64

	err := s.Db.QueryRow(ctx, `
			SELECT SUM(price)
			FROM subscriptions
			WHERE username = $1
				AND ($2::text IS NULL OR service_name = $2)
				AND start_date <= COALESCE($3, start_date)
				AND (end_date IS NULL OR end_date >= COALESCE($4, end_date))
		`,
		entry.Username, entry.ServiceName, entry.EndDate, entry.StartDate).Scan(&res)

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if !res.Valid {
		return 0, nil
	}

	return res.Float64, nil
}

func (s *Storage) RegisterUser(ctx context.Context, username, passwordHash string) error {
	const op = "storage.postgresql.RegisterUser"
	_, err := s.Db.Exec(ctx, `
		INSERT INTO users (
			username,
			password_hash
		) VALUES ($1, $2);`,
		username, passwordHash)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*user.User, error) {
	const op = "storage.postgresql.GetUserByUsername"
	user := &user.User{}
	row := s.Db.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users	
		WHERE username = $1
		`, username)
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return user, nil
}
