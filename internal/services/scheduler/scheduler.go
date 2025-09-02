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
	FindSubscriptionExpiringToday(ctx context.Context) ([]*models.User, error)
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
	s.runFindExpiringTrialPeriod(ctx, channel)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.runFindExpiringTrialPeriod(ctx, channel)
	}
}

func (s *SchedulerService) runFindExpiringTrialPeriod(ctx context.Context, channel *amqp.Channel) {
	s.log.Info("starting service to find expiring trial period for subscription")
	entriesInfo, err := s.repo.FindSubscriptionExpiringToday(ctx)
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
