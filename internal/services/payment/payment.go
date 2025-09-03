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
}

type SubscriptionService struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

func New(repo SubscriptionRepository, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{
		repo: repo,
		log:  log,
	}
}

func (s *SubscriptionService) GetOrCreatePaymentToken(ctx context.Context, userUID string, token string) (int, error) {
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

func (s *SubscriptionService) ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error) {
	return s.repo.ListPaymentTokens(ctx, userUID)
}
