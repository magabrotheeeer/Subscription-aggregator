package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/month"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

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
