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
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/month"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
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

// ===== SUBSCRIPTION METHODS =====

// CreateEntry вставляет новую запись подписки и возвращает её ID.
func (s *Storage) CreateEntry(ctx context.Context, entry models.Entry) (int, error) {
	const op = "storage.CreateEntry"
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `INSERT INTO subscriptions (service_name, price, username, start_date,
			      counter_months, user_uid, next_payment_date, is_active) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id`
	var newID int
	err := s.DB.QueryRowContext(ctx, query,
		entry.ServiceName, entry.Price, entry.Username, entry.StartDate, entry.CounterMonths,
		entry.UserUID, entry.NextPaymentDate, entry.IsActive).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// RemoveEntry удаляет подписку по ID и возвращает количество удалённых строк.
func (s *Storage) RemoveEntry(ctx context.Context, id int) (int, error) {
	const op = "storage.RemoveEntry"
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `DELETE FROM subscriptions WHERE id = $1`
	result, err := s.DB.ExecContext(ctx, query, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return int(rowsAffected), nil
}

// ReadEntry возвращает данные подписки по её ID.
func (s *Storage) ReadEntry(ctx context.Context, id int) (*models.Entry, error) {
	const op = "storage.ReadEntry"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT service_name, price, username, start_date, counter_months,
				user_uid, next_payment_date, is_active
			  FROM subscriptions WHERE id = $1`
	row := s.DB.QueryRowContext(ctx, query, id)

	var result models.Entry
	if err := row.Scan(&result.ServiceName, &result.Price, &result.Username, &result.StartDate,
		&result.CounterMonths, &result.UserUID, &result.NextPaymentDate, &result.IsActive); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &result, nil
}

// UpdateEntry обновляет данные подписки по её ID и возвращает количество изменённых строк.
func (s *Storage) UpdateEntry(ctx context.Context, req models.Entry, id int, username string) (int, error) {
	const op = "storage.UpdateEntry"
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `UPDATE subscriptions 
			  SET service_name = $1, price = $2, username = $3, start_date = $4, 
			      counter_months = $5, user_uid = $6, next_payment_date = $7, is_active = $8
			  WHERE id = $9`
	result, err := s.DB.ExecContext(ctx, query,
		req.ServiceName, req.Price, username, req.StartDate,
		req.CounterMonths, req.UserUID, req.NextPaymentDate, req.IsActive, id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return int(rowsAffected), nil
}

// ListEntrys возвращает список всех подписок пользователя с пагинацией.
func (s *Storage) ListEntrys(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error) {
	const op = "storage.ListEntrys"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT service_name, price, username, start_date, counter_months, user_uid, next_payment_date, is_active
			  FROM subscriptions
			  WHERE username = $1
			  ORDER BY id
			  LIMIT $2 OFFSET $3`
	rows, err := s.DB.QueryContext(ctx, query, username, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result []*models.Entry
	for rows.Next() {
		var item models.Entry
		if err := rows.Scan(&item.ServiceName, &item.Price, &item.Username, &item.StartDate,
			&item.CounterMonths, &item.UserUID, &item.NextPaymentDate, &item.IsActive); err != nil {
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

// CountSumEntrys подсчитывает суммарную стоимость подписок пользователя за выбранный период с учётом фильтров.
func (s *Storage) CountSumEntrys(ctx context.Context, entry models.FilterSum) (float64, error) {
	const op = "storage.CountSumEntrys"
	select {
	case <-ctx.Done():
		return 0.0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	filterEnd := entry.StartDate.AddDate(0, entry.CounterMonths, 0)

	query := `SELECT service_name, price, start_date, counter_months
              FROM subscriptions
              WHERE username = $1
		      	AND is_active = true	
          		AND ($2::text IS NULL OR service_name = $2)
          		AND start_date < $3
          		AND (start_date + (counter_months || ' months')::interval) > $4`
	rows, err := s.DB.QueryContext(ctx, query, entry.Username, entry.ServiceName, filterEnd, entry.StartDate)

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

// ListAllEntrys возвращает список всех подписок с пагинацией.
func (s *Storage) ListAllEntrys(ctx context.Context, limit, offset int) ([]*models.Entry, error) {
	const op = "storage.ListAllEntrys"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT service_name, price, username, start_date, counter_months, user_uid,
			      next_payment_date, is_active
			  FROM subscriptions
		      LIMIT $1 OFFSET $2`
	rows, err := s.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var result []*models.Entry
	for rows.Next() {
		var item models.Entry
		if err := rows.Scan(&item.ServiceName, &item.Price, &item.Username, &item.StartDate,
			&item.CounterMonths, &item.UserUID, &item.NextPaymentDate, &item.IsActive); err != nil {
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

// FindSubscriptionExpiringTomorrow находит подписки, истекающие завтра
func (s *Storage) FindSubscriptionExpiringTomorrow(ctx context.Context) ([]*models.EntryInfo, error) {
	const op = "storage.FindSubscriptionExpiringTomorrow"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT
		          u.email,
			      s.username,
			      s.service_name,
			      (s.start_date + (s.counter_months || ' months')::INTERVAL)::DATE AS end_date,
			      s.price
			  FROM subscriptions s
		      JOIN users u ON s.username = u.username
		      WHERE (s.start_date + (s.counter_months || ' months')::INTERVAL)::DATE = CURRENT_DATE + INTERVAL '1 day';`
	rows, err := s.DB.QueryContext(ctx, query)
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

// FindOldNextPaymentDate находит подписки со старыми датами платежей
func (s *Storage) FindOldNextPaymentDate(ctx context.Context) ([]*models.Entry, error) {
	const op = "storage.FindOldNextPaymentDate"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}
	query := `SELECT id, service_name, price, username, 
			    start_date, counter_months, user_uid, next_payment_date, is_active
			  FROM subscriptions
			  WHERE next_payment_date < CURRENT_DATE
			  AND is_active = true`

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		_ = rows.Close()
	}()
	var result []*models.Entry
	for rows.Next() {
		var item models.Entry
		if err := rows.Scan(&item.ID, &item.ServiceName, &item.Price, &item.Username, &item.StartDate,
			&item.CounterMonths, &item.UserUID, &item.NextPaymentDate, &item.IsActive); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil
}

// UpdateNextPaymentDate обновляет дату следующего платежа
func (s *Storage) UpdateNextPaymentDate(ctx context.Context, entry *models.Entry) (int, error) {
	const op = "storage.UpdateNextPaymentDate"
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `UPDATE subscriptions
		      SET next_payment_date = $1
		      WHERE id = $2`
	res, err := s.DB.ExecContext(ctx, query, entry.NextPaymentDate, entry.ID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return int(rowsAffected), nil
}

// GetActiveSubscriptionIDByUserUID получает ID активной подписки пользователя
func (s *Storage) GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID string, serviceName string) (string, error) {
	const op = "storage.GetActiveSubscriptionIDByUserUID"
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT id
		      FROM subscriptions
			  WHERE user_uid = $1
		  	    AND service_name = $2`
	var res string
	row := s.DB.QueryRowContext(ctx, query, userUID, serviceName)
	if err := row.Scan(&res); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return res, nil
}

// ===== USER METHODS =====

// RegisterUser сохраняет нового пользователя в базу данных и возвращает его ID.
func (s *Storage) RegisterUser(ctx context.Context, user models.User) (string, error) {
	const op = "storage.RegisterUser"
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	var newID string
	query := `INSERT INTO users (email, username, password_hash, role, trial_end_date,
			      subscription_status) 
			  VALUES ($1, $2, $3, $4, $5, $6)
			  RETURNING uid;`
	if err := s.DB.QueryRowContext(ctx, query,
		user.Email, user.Username, user.PasswordHash, user.Role, user.TrialEndDate,
		user.SubscriptionStatus).Scan(&newID); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// GetUserByUsername возвращает пользователя по его username.
func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	const op = "storage.GetUserByUsername"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT uid, email, username, password_hash, role, trial_end_date,
			      subscription_status, subscription_expiry
			  FROM users
			  WHERE username = $1`
	u := &models.User{}
	row := s.DB.QueryRowContext(ctx, query, username)

	var trialEndDate, subscriptionExpiry sql.NullTime
	if err := row.Scan(&u.UUID, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &trialEndDate, &u.SubscriptionStatus, &subscriptionExpiry); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if trialEndDate.Valid {
		u.TrialEndDate = &trialEndDate.Time
	}
	if subscriptionExpiry.Valid {
		u.SubscriptionExpire = &subscriptionExpiry.Time
	}
	return u, nil
}

// GetUser возвращает пользователя по его UID.
func (s *Storage) GetUser(ctx context.Context, userUID string) (*models.User, error) {
	const op = "storage.GetUser"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT uid, email, username, password_hash, role, trial_end_date,
			      subscription_status, subscription_expiry
			  FROM users
			  WHERE uid = $1`
	u := &models.User{}
	row := s.DB.QueryRowContext(ctx, query, userUID)

	var trialEndDate, subscriptionExpiry sql.NullTime
	if err := row.Scan(&u.UUID, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &trialEndDate, &u.SubscriptionStatus, &subscriptionExpiry); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if trialEndDate.Valid {
		u.TrialEndDate = &trialEndDate.Time
	}
	if subscriptionExpiry.Valid {
		u.SubscriptionExpire = &subscriptionExpiry.Time
	}
	return u, nil
}

// FindSubscriptionExpiringToday находит пользователей с истекающим сегодня пробным периодом
func (s *Storage) FindSubscriptionExpiringToday(ctx context.Context) ([]*models.User, error) {
	const op = "storage.FindSubscriptionExpiringToday"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT
			      uid, email, username, password_hash, role, trial_end_date,
			      subscription_status, subscription_expiry
			  FROM users
		      WHERE trial_end_date::DATE = CURRENT_DATE;`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var result []*models.User
	for rows.Next() {
		var u models.User
		var trialEndDate, subscriptionExpiry sql.NullTime
		if err = rows.Scan(&u.UUID, &u.Email, &u.Username, &u.PasswordHash,
			&u.Role, &trialEndDate, &u.SubscriptionStatus, &subscriptionExpiry,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		if trialEndDate.Valid {
			u.TrialEndDate = &trialEndDate.Time
		}
		if subscriptionExpiry.Valid {
			u.SubscriptionExpire = &subscriptionExpiry.Time
		}
		result = append(result, &u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil
}

// UpdateStatusActiveForSubscription обновляет статус подписки на активный
func (s *Storage) UpdateStatusActiveForSubscription(ctx context.Context, userUID, status string) error {
	const op = "storage.UpdateStatusActiveForSubscription"
	select {
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `UPDATE users
		      SET subscription_status = $1,
			      subscription_expiry = subscription_expiry + INTERVAL '1 month'
			  WHERE uid = $2`
	_, err := s.DB.ExecContext(ctx, query, status, userUID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// UpdateStatusCancelForSubscription обновляет статус подписки на отмененный
func (s *Storage) UpdateStatusCancelForSubscription(ctx context.Context, userUID, status string) error {
	const op = "storage.UpdateStatusCancelForSubscription"
	select {
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `UPDATE users
			  SET subscription_status = $1
		      WHERE uid = $2`
	_, err := s.DB.ExecContext(ctx, query, status, userUID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetSubscriptionStatus получает статус подписки пользователя
func (s *Storage) GetSubscriptionStatus(ctx context.Context, userUID string) (bool, error) {
	const op = "storage.GetSubscriptionStatus"
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT is_active FROM subscriptions WHERE user_uid = $1 LIMIT 1`
	var isActive bool
	err := s.DB.QueryRowContext(ctx, query, userUID).Scan(&isActive)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return isActive, nil
}

// ===== PAYMENT METHODS =====

// FindPaymentToken находит токен платежа
func (s *Storage) FindPaymentToken(ctx context.Context, userUID string, token string) (int, bool, error) {
	const op = "storage.FindPaymentToken"
	select {
	case <-ctx.Done():
		return 0, false, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT id FROM yookassa_payment_tokens 
			  WHERE user_uid = $1 AND token = $2`
	var id int
	err := s.DB.QueryRowContext(ctx, query, userUID, token).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("%s: %w", op, err)
	}
	return id, true, nil
}

// CreatePaymentToken создает новый токен платежа
func (s *Storage) CreatePaymentToken(ctx context.Context, userUID string, token string) (int, error) {
	const op = "storage.CreatePaymentToken"
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `INSERT INTO yookassa_payment_tokens (user_uid, token) 
			  VALUES ($1, $2) RETURNING id`
	var newID int
	err := s.DB.QueryRowContext(ctx, query, userUID, token).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}

// ListPaymentTokens возвращает список токенов платежей пользователя
func (s *Storage) ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error) {
	const op = "storage.ListPaymentTokens"
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `SELECT id, user_uid, token, created_at 
			  FROM yookassa_payment_tokens 
		      WHERE user_uid = $1`
	rows, err := s.DB.QueryContext(ctx, query, userUID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var result []*models.PaymentToken
	for rows.Next() {
		var pt models.PaymentToken
		if err := rows.Scan(&pt.ID, &pt.UserUID, &pt.Token, &pt.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &pt)
	}
	return result, nil
}

// SavePayment сохраняет информацию о платеже
func (s *Storage) SavePayment(ctx context.Context, payload *paymentwebhook.Payload, amount int64, userUID string) (int, error) {
	const op = "storage.SavePayment"
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	query := `INSERT INTO yookassa_payments (user_uid, payment_id, status, amount, currency, created_at) 
			  VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id`
	var newID int
	err := s.DB.QueryRowContext(ctx, query,
		userUID, payload.Object.ID, payload.Object.Status, amount,
		payload.Object.Amount.Currency).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return newID, nil
}
