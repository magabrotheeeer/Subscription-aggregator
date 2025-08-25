package services

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type SenderService struct {
	log *slog.Logger
}

// NewSenderService создает новый экземпляр SenderService.
func NewSenderService(log *slog.Logger) *SenderService {
	return &SenderService{
		log: log,
	}
}

func (s *SenderService) SendInfoExpiringSubscription(body []byte) error {
	var message models.EntryInfo
	if err := json.Unmarshal(body, &message); err != nil {
		return fmt.Errorf("error unmarshalling message: %w", err)
	}
	//TODO: отправка сообщения по email, которое уведомляет о скором окончании подписки
	return nil

}
