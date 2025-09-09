package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type SubscriptionRepository interface {
	FindPaymentToken(ctx context.Context, userUID string, token string) (int, bool, error)
	CreatePaymentToken(ctx context.Context, userUID string, token string) (int, error)
	ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error)
	GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID, serviceName string) (string, error)
}

type PaymentService struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

func New(repo SubscriptionRepository, log *slog.Logger) *PaymentService {
	return &PaymentService{
		repo: repo,
		log:  log,
	}
}

func (s *PaymentService) GetOrCreatePaymentToken(ctx context.Context, userUID string, token string) (int, error) {
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

func (s *PaymentService) ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error) {
	return s.repo.ListPaymentTokens(ctx, userUID)
}

func (s *PaymentService) GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID string) (string, error) {
	serviceName := "Subscription-Aggregator"
	return s.repo.GetActiveSubscriptionIDByUserUID(ctx, userUID, serviceName)
}
