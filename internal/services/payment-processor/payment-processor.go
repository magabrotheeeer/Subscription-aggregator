package paymentprocessor

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// TODO
type PaymentProvider interface {
}

// TODO
type PaymentRepository interface {
}

type PaymentService struct {
	paymentProvider PaymentProvider   // интерфейс для платежного шлюза
	repo            PaymentRepository // для работы с БД платежей и подписок
	log             *slog.Logger
}

func NewPaymentService(provider PaymentProvider, repo PaymentRepository, logger *slog.Logger) *PaymentService {
	return &PaymentService{
		paymentProvider: provider,
		repo:            repo,
		log:             logger,
	}
}

func (s *PaymentService) ProcessSubscriptionPayment(body []byte) error {
	var message models.Entry
	if err := json.Unmarshal(body, &message); err != nil {
		s.log.Error("Failed to unmarshal message body", "error", sl.Err(err))
		return fmt.Errorf("error unmarshalling message: %w", err)
	}
	//TODO валидация данных подписки???
	//TODO вызов платежного провайдера
	//TODO обработка результатов платежа

	return nil
}
