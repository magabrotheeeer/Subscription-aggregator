package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/rabbitmq"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/streadway/amqp"
)

type SubscriptionRepository interface {
	FindSubscriptionExpiringTomorrow(ctx context.Context) ([]*models.EntryInfo, error)
	FindSubscriptionsDueToday(ctx context.Context) ([]*models.Entry, error)
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

func (s *SchedulerService) FindExpiringSubscriptionsDueTomorrow(ctx context.Context, channel *amqp.Channel) {
	s.runFindExpiringSubscriptionsDueTomorrow(ctx, channel)

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.runFindExpiringSubscriptionsDueTomorrow(ctx, channel)
	}
}

func (s *SchedulerService) runFindExpiringSubscriptionsDueTomorrow(ctx context.Context, channel *amqp.Channel) {
	s.log.Info("starting service to find expiring subscriptions due tomorrow")
	entriesInfo, err := s.repo.FindSubscriptionExpiringTomorrow(ctx)
	if err != nil {
		s.log.Error("failed to find entries", sl.Err(err))
		return
	}
	if len(entriesInfo) == 0 {
		s.log.Info("no expiring subscriptions found")
		return
	}
	s.log.Info("found expiring subscriptions", "count", len(entriesInfo))
	for _, entryInfo := range entriesInfo {
		err = rabbitmq.PublishMessage(channel, "notifications", "upcoming", entryInfo)
		if err != nil {
			s.log.Error("failed to publish message", sl.Err(err))
		}
	}
}

func (s *SchedulerService) FindExpiringSubscriptionsDueToday(ctx context.Context, channel *amqp.Channel) {
	s.runFindExpiringSubscriptionsDueToday(ctx, channel)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.runFindExpiringSubscriptionsDueToday(ctx, channel)
	}
}

func (s *SchedulerService) runFindExpiringSubscriptionsDueToday(ctx context.Context, channel *amqp.Channel) {
	s.log.Info("starting service to find expiring subscriptions due today")
	entriesInfo, err := s.repo.FindSubscriptionsDueToday(ctx)
	if err != nil {
		s.log.Error("failed to find entries", sl.Err(err))
		return
	}
	if len(entriesInfo) == 0 {
		s.log.Info("no subscriptions for pay found")
		return
	}
	s.log.Info("found subscriptions for pay", "count", len(entriesInfo))
	for _, entryInfo := range entriesInfo {
		err = rabbitmq.PublishMessage(channel, "payment", "due", entryInfo)
		if err != nil {
			s.log.Error("failed to publish message", sl.Err(err))
		}
	}

}
