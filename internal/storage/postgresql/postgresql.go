// Package postgresql реализует хранилище данных на основе PostgreSQL
// для управления подписками и пользователями. Предоставляет методы
// создания, чтения, обновления, удаления и агрегирования записей,
// а также работу с пользователями.
package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/count_sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/month"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Storage инкапсулирует соединение с базой данных PostgreSQL
// и реализует методы работы с подписками и пользователями.
type Storage struct {
	Db *pgx.Conn
}

// New создаёт подключение к PostgreSQL и инициализирует необходимые таблицы и индексы.
func New(storageConnectionString string) (*Storage, error) {
	const op = "storage.postgresql.New"

	conn, err := pgx.Connect(context.Background(), storageConnectionString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Создание таблицы подписок
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS subscriptions(
			id SERIAL PRIMARY KEY,
			service_name TEXT NOT NULL,
			price NUMERIC(10, 2) NOT NULL,
			username VARCHAR(255) NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE
		);`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Индекс по username для быстрого поиска подписок пользователя
	_, err = conn.Exec(context.Background(), `
		CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id 
		ON subscriptions (username);`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Таблица пользователей
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users(
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Индекс по username для таблицы пользователей
	_, err = conn.Exec(context.Background(), `
		CREATE INDEX IF NOT EXISTS idx_users_username
		ON users (username);`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{Db: conn}, nil
}

// CreateSubscriptionEntry вставляет новую запись подписки и возвращает её ID.
func (s *Storage) CreateSubscriptionEntry(ctx context.Context, entry models.Entry) (int, error) {
	const op = "storage.postgresql.CreateSubscriptionEntry"
	var newID int
	err := s.Db.QueryRow(ctx, `
			INSERT INTO subscriptions (service_name, price, username, start_date, end_date) 
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
		entry.ServiceName, entry.Price, entry.Username, entry.StartDate, entry.EndDate).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// RemoveSubscriptionEntry удаляет подписку по ID и возвращает количество удалённых строк.
func (s *Storage) RemoveSubscriptionEntry(ctx context.Context, id int) (int64, error) {
	const op = "storage.postgresql.DeleteSubscriptionEntryByUserID"
	commandTag, err := s.Db.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return commandTag.RowsAffected(), nil
}

// ReadSubscriptionEntry возвращает данные подписки по её ID.
func (s *Storage) ReadSubscriptionEntry(ctx context.Context, id int) (*models.Entry, error) {
	const op = "storage.postgresql.ReadSubscriptionEntryByUserID"
	row := s.Db.QueryRow(ctx, `
		SELECT service_name, price, username, start_date, end_date 
		FROM subscriptions WHERE id = $1`, id)

	var result models.Entry
	if err := row.Scan(&result.ServiceName, &result.Price, &result.Username, &result.StartDate, &result.EndDate); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &result, nil
}

// UpdateSubscriptionEntry обновляет данные подписки по её ID и возвращает количество изменённых строк.
func (s *Storage) UpdateSubscriptionEntry(ctx context.Context, entry models.Entry, id int) (int64, error) {
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
	return commandTag.RowsAffected(), nil
}

// ListSubscriptionEntrys возвращает список всех подписок пользователя с пагинацией.
func (s *Storage) ListSubscriptionEntrys(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error) {
	const op = "storage.postgresql.ListSubscriptionEntrys"
	rows, err := s.Db.Query(ctx, `
		SELECT service_name, price, username, start_date, end_date
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
		if err := rows.Scan(&item.ServiceName, &item.Price, &item.Username, &item.StartDate, &item.EndDate); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	return result, nil
}

// CountSumSubscriptionEntrys считает суммарную стоимость подписок пользователя за выбранный период с учётом фильтров.
func (s *Storage) CountSumSubscriptionEntrys(ctx context.Context, entry countsum.FilterSum) (float64, error) {
	const op = "storage.postgresql.CountSumSubscriptionEntrys"

	rows, err := s.Db.Query(ctx, `
		SELECT service_name, price, start_date, end_date
		FROM subscriptions
		WHERE username = $1
			AND ($2::text IS NULL OR service_name = $2)
			AND start_date <= COALESCE($3, start_date)
			AND (end_date IS NULL OR end_date > COALESCE($4, end_date))
`, entry.Username, entry.ServiceName, entry.EndDate, entry.StartDate)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var total float64
	for rows.Next() {
		var serviceName string
		var price float64
		var startDate time.Time
		var endDate *time.Time

		if err := rows.Scan(&serviceName, &price, &startDate, &endDate); err != nil {
			return 0, fmt.Errorf("%s: %w", op, err)
		}

		// Начало периода активности
		activeStart := month.MaxDate(startDate, entry.StartDate)

		// Конец фильтра
		filterEnd := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		if entry.EndDate != nil {
			filterEnd = *entry.EndDate
		}

		// Коррекция даты окончания подписки (-1 день)
		var adjustedEndDate *time.Time
		if endDate != nil {
			newEnd := endDate.AddDate(0, 0, -1)
			adjustedEndDate = &newEnd
		}

		// Определяем финальную дату окончания активности
		var activeEnd time.Time
		if adjustedEndDate != nil && adjustedEndDate.Before(filterEnd) {
			activeEnd = *adjustedEndDate
		} else {
			activeEnd = filterEnd
		}

		// Если период валиден — считаем стоимость
		if !activeEnd.Before(activeStart) {
			months := month.MonthsBetween(activeStart, activeEnd)
			total += price * float64(months)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return total, nil
}

// RegisterUser сохраняет нового пользователя в базу данных и возвращает его ID.
func (s *Storage) RegisterUser(ctx context.Context, username, passwordHash string) (int, error) {
	const op = "storage.postgresql.RegisterUser"
	var newID int
	if err := s.Db.QueryRow(ctx, `
			INSERT INTO users (username, password_hash) 
			VALUES ($1, $2)
			RETURNING id;`,
		username, passwordHash).Scan(&newID); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// GetUserByUsername возвращает пользователя по его username.
func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	const op = "storage.postgresql.GetUserByUsername"
	u := &models.User{}
	row := s.Db.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE username = $1`, username)

	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return u, nil
}
