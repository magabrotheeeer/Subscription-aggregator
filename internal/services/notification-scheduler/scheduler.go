package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	"github.com/streadway/amqp"
)

type SubscriptionRepository interface {
	FindSubscriptionExpiringTomorrow(ctx context.Context) ([]*models.EntryInfo, error)
}

type SchedulerService struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

// NewSchedulerService создает новый экземпляр SchedulerService.
func NewSchedulerService(repo SubscriptionRepository, log *slog.Logger) *SchedulerService {
	return &SchedulerService{
		repo: repo,
		log:  log,
	}
}

func (s *SchedulerService) FindExpiringSubscriptions(ctx context.Context, channel *amqp.Channel) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.log.Info("starting service to find expiring subscriptions")
		entriesInfo, err := s.repo.FindSubscriptionExpiringTomorrow(ctx)
		if err != nil {
			s.log.Error("failed to find entries", sl.Err(err))
		}
		for _, entryInfo := range entriesInfo {
			err = rabbitmq.PublishMessage(channel, "notifications", "upcoming", entryInfo)
			if err != nil {
				s.log.Error("failed to publish message", sl.Err(err))
			}
		}
	}

}
