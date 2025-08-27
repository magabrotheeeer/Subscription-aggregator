package paymentmethods

import (
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/paymentprovider"
)

// TODO: получаем данные из хэндлера и добавляем в бд

type PaymentMethodsRepository interface {
	//TODO: функции для обращения к бд
}

type PaymentMethodsService struct {
	repo   PaymentMethodsRepository
	client paymentprovider.Client
	log    *slog.Logger
}

func NewPaymentMethodsService(repo PaymentMethodsRepository, log *slog.Logger) *PaymentMethodsService {
	return &PaymentMethodsService{
		repo: repo,
		log:  log,
	}
}

func (p *PaymentMethodsService) CreatePaymentMethod(paymentResponse paymentprovider.CreatePaymentMethodResponse) {

}