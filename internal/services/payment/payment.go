// Package payment предоставляет сервис для работы с платежами.
package payment

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// SubscriptionRepository определяет интерфейс для работы с подписками в репозитории.
type SubscriptionRepository interface {
	FindPaymentToken(ctx context.Context, userUID string, token string) (int, bool, error)
	CreatePaymentToken(ctx context.Context, userUID string, token string) (int, error)
	ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error)
	GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID, serviceName string) (string, error)
	SavePayment(ctx context.Context, payload *paymentwebhook.Payload, amount int64, userUID string) (int, error)
	UpdateStatusActiveForSubscription(ctx context.Context, userUID, status string) error
	UpdateStatusCancelForSubscription(ctx context.Context, userUID, status string) error
}

// Service предоставляет сервис для работы с платежами.
type Service struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

// New создает новый экземпляр Service.
func New(repo SubscriptionRepository, log *slog.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

// GetOrCreatePaymentToken получает или создает токен платежного метода.
func (s *Service) GetOrCreatePaymentToken(ctx context.Context, userUID string, token string) (int, error) {
	res, found, err := s.repo.FindPaymentToken(ctx, userUID, token)
	if err != nil {
		return 0, fmt.Errorf("failed to find token: %w", err)
	}
	if found {
		return res, nil
	}
	res, err = s.repo.CreatePaymentToken(ctx, userUID, token)
	if err != nil {
		return 0, fmt.Errorf("failed to create token: %w", err)
	}
	return res, nil
}

// ListPaymentTokens возвращает список токенов платежных методов пользователя.
func (s *Service) ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error) {
	return s.repo.ListPaymentTokens(ctx, userUID)
}

// GetActiveSubscriptionIDByUserUID возвращает ID активной подписки пользователя.
func (s *Service) GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID string) (string, error) {
	serviceName := "Subscription-Aggregator"
	return s.repo.GetActiveSubscriptionIDByUserUID(ctx, userUID, serviceName)
}

// SavePayment сохраняет информацию о платеже.
func (s *Service) SavePayment(ctx context.Context, payload *paymentwebhook.Payload) (int, error) {
	userUID, exists := payload.Object.Metadata["user_uid"]
	if !exists || userUID == "" {
		return 0, fmt.Errorf("user_uid not found in metadata")
	}

	amount, err := strconv.ParseFloat(payload.Object.Amount.Value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}
	amountInKopecks := int64(amount * 100)
	return s.repo.SavePayment(ctx, payload, amountInKopecks, userUID)
}

// UpdateStatusActiveForSubscription обновляет статус подписки на активный.
func (s *Service) UpdateStatusActiveForSubscription(ctx context.Context, userUID string) error {
	return s.repo.UpdateStatusActiveForSubscription(ctx, userUID, "active")
}

// UpdateStatusExpireForSubscription обновляет статус подписки на истекший.
func (s *Service) UpdateStatusExpireForSubscription(ctx context.Context, userUID string) error {
	return s.repo.UpdateStatusCancelForSubscription(ctx, userUID, "expire")
}

// UpdateStatusCancelForSubscription обновляет статус подписки на отмененный.
func (s *Service) UpdateStatusCancelForSubscription(ctx context.Context, userUID string) error {
	return s.repo.UpdateStatusCancelForSubscription(ctx, userUID, "cancel")
}
