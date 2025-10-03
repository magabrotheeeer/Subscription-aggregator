package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

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
