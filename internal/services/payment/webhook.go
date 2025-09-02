package payment

import (
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/payment/paymentwebhook"
)

type SubscriptionRepository interface {
}

type PaymentService struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

func New(log *slog.Logger, repo SubscriptionRepository) *PaymentService {
	return &PaymentService{
		log:  log,
		repo: repo,
	}
}

func (ps *PaymentService) CreatePayment(payload *paymentwebhook.Payload) (int, error){
	
}

