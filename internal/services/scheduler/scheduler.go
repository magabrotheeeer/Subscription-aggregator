package services

import (
	"context"
	"fmt"
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
	FindOldNextPaymentDate(ctx context.Context) ([]*models.Entry, error)
	UpdateNextPaymentDate(ctx context.Context, entry *models.Entry) (int, error)
}

type Cache interface {
	// Set сохраняет значение в кеш с временем жизни.
	Set(key string, value any, expiration time.Duration) error
}

type SchedulerService struct {
	repo  SubscriptionRepository
	cache Cache
	log   *slog.Logger
}

// NewSchedulerService создает новый экземпляр SchedulerService.
func NewSchedulerService(repo SubscriptionRepository, cache Cache, log *slog.Logger) *SchedulerService {
	return &SchedulerService{
		repo:  repo,
		cache: cache,
		log:   log,
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
		s.log.Info("no expiring subscriptions due tomorrow found")
		return
	}
	s.log.Info("found expiring subscriptions", "count", len(entriesInfo))
	for _, entryInfo := range entriesInfo {
		err = rabbitmq.PublishMessage(channel, "notifications", "subscription.expiring.tomorrow", entryInfo)
		if err != nil {
			s.log.Error("failed to publish message", sl.Err(err))
		}
	}
	s.log.Info("success to publish all messages")
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
		s.log.Info("no expiring trial period subscriptions found")
		return
	}
	s.log.Info("found expiring subscriptions", "count", len(entriesInfo))
	for _, entryInfo := range entriesInfo {
		err = rabbitmq.PublishMessage(channel, "notifications", "subscription.trial.expiring", entryInfo)
		if err != nil {
			s.log.Error("failed to publish message", sl.Err(err))
		}
	}
	s.log.Info("success to publish all messages")
}

func (s *SchedulerService) FindOldNextPaymentDate(ctx context.Context) {
	s.runFindOldNextPaymentDate(ctx)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.runFindOldNextPaymentDate(ctx)
	}
}

func (s *SchedulerService) runFindOldNextPaymentDate(ctx context.Context) {
	s.log.Info("starting worker which updates next payment date")
	entriesInfo, err := s.repo.FindOldNextPaymentDate(ctx)
	if err != nil {
		s.log.Error("failed to find entries", sl.Err(err))
		return
	}
	if len(entriesInfo) == 0 {
		s.log.Info("all entrys are up to date")
	}
	s.log.Info("outdated next payment dates found")
	for _, entryInfo := range entriesInfo {
		current := entryInfo.NextPaymentDate
		newDate := current.AddDate(0, 1, 0)
		entryInfo.NextPaymentDate = newDate
		id, err := s.repo.UpdateNextPaymentDate(ctx, entryInfo)
		if err != nil {
			s.log.Error("failed to update next payment date",
				slog.Int("id", id),
				sl.Err(err))
			continue
		}
		cacheKey := fmt.Sprintf("subscription:%d", id)
		if err := s.cache.Set(cacheKey, entryInfo, time.Hour); err != nil {
			s.log.Warn("failed to cache subscription", slog.String("key", cacheKey), sl.Err(err))
		}
	}
	s.log.Info("success to update")
}
