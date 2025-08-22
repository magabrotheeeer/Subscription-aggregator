// Package storage реализует хранилище данных на основе PostgreSQL
// для управления подписками и пользователями. Предоставляет методы
// создания, чтения, обновления, удаления и агрегирования записей,
// а также работу с пользователями.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Регистрация драйвера pgx для использования с database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/month"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Storage инкапсулирует соединение с базой данных PostgreSQL
// и реализует методы работы с подписками и пользователями.
type Storage struct {
	Db *sql.DB
}

// New создаёт подключение к PostgreSQL и инициализирует необходимые таблицы и индексы.
func New(storageConnectionString string) (*Storage, error) {
	const op = "storage.postgresql.New"

	db, err := sql.Open("pgx", storageConnectionString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err = db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{Db: db}, nil
}

// Create вставляет новую запись подписки и возвращает её ID.
func (s *Storage) Create(ctx context.Context, entry models.Entry) (int, error) {
	const op = "storage.postgresql.Create"
	var newID int
	err := s.Db.QueryRowContext(ctx, `
			INSERT INTO subscriptions (service_name, price, username, start_date, counter_months) 
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
		entry.ServiceName, entry.Price, entry.Username, entry.StartDate, entry.CounterMonths).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// Remove удаляет подписку по ID и возвращает количество удалённых строк.
func (s *Storage) Remove(ctx context.Context, id int) (int, error) {
	const op = "storage.postgresql.Remove"
	result, err := s.Db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return int(rowsAffected), nil
}

// Read возвращает данные подписки по её ID.
func (s *Storage) Read(ctx context.Context, id int) (*models.Entry, error) {
	const op = "storage.postgresql.Read"
	row := s.Db.QueryRowContext(ctx, `
		SELECT service_name, price, username, start_date, counter_months
		FROM subscriptions WHERE id = $1`, id)

	var result models.Entry
	if err := row.Scan(&result.ServiceName, &result.Price, &result.Username, &result.StartDate, &result.CounterMonths); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &result, nil
}

// Update обновляет данные подписки по её ID и возвращает количество изменённых строк.
func (s *Storage) Update(ctx context.Context, entry models.Entry, id int) (int, error) {
	const op = "storage.postgresql.Update"
	result, err := s.Db.ExecContext(ctx, `
		UPDATE subscriptions
		SET service_name = $1,
			start_date = $2,
			counter_months = $3,
			price = $4,
			username = $5
		WHERE id = $6`,
		entry.ServiceName, entry.StartDate, entry.CounterMonths, entry.Price, entry.Username, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return int(rowsAffected), nil
}

// List возвращает список всех подписок пользователя с пагинацией.
func (s *Storage) List(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error) {
	const op = "storage.postgresql.List"
	rows, err := s.Db.QueryContext(ctx, `
		SELECT service_name, price, username, start_date, counter_months
		FROM subscriptions
		WHERE username = $3
		LIMIT $1 OFFSET $2`,
		limit, offset, username)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result []*models.Entry
	for rows.Next() {
		var item models.Entry
		if err := rows.Scan(&item.ServiceName, &item.Price, &item.Username, &item.StartDate, &item.CounterMonths); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil
}

// CountSum подсчитывает суммарную стоимость подписок пользователя за выбранный период с учётом фильтров.
func (s *Storage) CountSum(ctx context.Context, entry models.FilterSum) (float64, error) {
	const op = "storage.postgresql.CountSum"
	filterEnd := entry.StartDate.AddDate(0, entry.CounterMonths, 0)

	rows, err := s.Db.QueryContext(ctx, `
        SELECT service_name, price, start_date, counter_months
        FROM subscriptions
        WHERE username = $1
          AND ($2::text IS NULL OR service_name = $2)
          AND start_date < $3
          AND (start_date + (counter_months || ' months')::interval) > $4
    `, entry.Username, entry.ServiceName, filterEnd, entry.StartDate)

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	var total float64
	for rows.Next() {
		var serviceName string
		var price float64
		var startDate time.Time
		var counterMonths int

		if err := rows.Scan(&serviceName, &price, &startDate, &counterMonths); err != nil {
			return 0, fmt.Errorf("%s: %w", op, err)
		}

		remainingMonths := month.CountMonths(startDate, counterMonths, entry.StartDate)
		total += price * float64(remainingMonths)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	err = rows.Close()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return total, nil
}

// ListAll возвращает список всех подписок с пагинацией.
func (s *Storage) ListAll(ctx context.Context, limit, offset int) ([]*models.Entry, error) {
	const op = "storage.postgresql.ListAll"
	rows, err := s.Db.QueryContext(ctx, `
		SELECT service_name, price, username, start_date, counter_months
		FROM subscriptions
		LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result []*models.Entry
	for rows.Next() {
		var item models.Entry
		if err := rows.Scan(&item.ServiceName, &item.Price, &item.Username, &item.StartDate, &item.CounterMonths); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}

	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}

// RegisterUser сохраняет нового пользователя в базу данных и возвращает его ID.
func (s *Storage) RegisterUser(ctx context.Context, user models.User) (int, error) {
	const op = "storage.postgresql.RegisterUser"
	var newID int
	if err := s.Db.QueryRowContext(ctx, `
			INSERT INTO users (email, username, password_hash, role) 
			VALUES ($1, $2, $3, $4)
			RETURNING id;`,
		user.Email, user.Username, user.PasswordHash, user.Role).Scan(&newID); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// GetUserByUsername возвращает пользователя по его username.
func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	const op = "storage.postgresql.GetUserByUsername"
	u := &models.User{}
	row := s.Db.QueryRowContext(ctx, `
		SELECT id, email, username, password_hash, role
		FROM users
		WHERE username = $1`, username)

	if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return u, nil
}

func (s *Storage) FindSubscriptionExpiringTomorrow(ctx context.Context) ([]*models.EntryInfo, error) {
	const op = "storage.postgresql.FindSubscriptionExpiringTomorrow"
	rows, err := s.Db.QueryContext(ctx, `
		SELECT
			u.email,
			s.username,
			s.service_name,
			(s.start_date + (s.counter_months || ' months')::INTERVAL)::DATE AS end_date,
			s.price
		FROM subscriptions s
		JOIN users u ON s.username = u.username
		WHERE (s.start_date + (s.counter_months || ' months')::INTERVAL)::DATE = CURRENT_DATE + INTERVAL '1 day';
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var result []*models.EntryInfo
	for rows.Next() {
		var si models.EntryInfo
		if err = rows.Scan(&si.Email, &si.Username, &si.ServiceName,
			&si.EndDate, &si.Price); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &si)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil
}
