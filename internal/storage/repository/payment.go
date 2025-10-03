package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

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
