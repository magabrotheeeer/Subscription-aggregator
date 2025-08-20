package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub models.Entry) (int, error)
	Remove(ctx context.Context, id int) (int, error)
	Read(ctx context.Context, id int) (*models.Entry, error)
	Update(ctx context.Context, entry models.Entry, id int) (int, error)
	List(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error)
	CountSum(ctx context.Context, entry models.FilterSum) (float64, error)
	ListAll(ctx context.Context, limit, offset int) ([]*models.Entry, error)
}

type Cache interface {
	Get(key string, result any) (bool, error)
	Set(key string, value any, expiration time.Duration) error
	Invalidate(key string) error
}

type SubscriptionService struct {
	repo  SubscriptionRepository
	cache Cache
	log   *slog.Logger
}

func NewSubscriptionService(repo SubscriptionRepository, cache Cache, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

func (s *SubscriptionService) Create(ctx context.Context, userName string, req models.DummyEntry) (int, error) {
	startDate, err := time.Parse("02-01-2006", req.StartDate)
	if err != nil {
		return 0, fmt.Errorf("invalid start date: %w", err)
	}

	entry := models.Entry{
		ServiceName:   req.ServiceName,
		Username:      userName,
		Price:         req.Price,
		StartDate:     startDate,
		CounterMonths: req.CounterMonths,
	}

	id, err := s.repo.Create(ctx, entry)
	if err != nil {
		return 0, err
	}

	s.log.Info("created new subscription", slog.Int("id", id))

	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Set(cacheKey, entry, time.Hour); err != nil {
		s.log.Warn("failed to cache subscription", slog.String("key", cacheKey), slog.Any("err", err))
	}
	s.log.Info("created new susbcription in cache")

	return id, nil
}

func (s *SubscriptionService) Remove(ctx context.Context, id int) (int, error) {
	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Invalidate(cacheKey); err != nil {
		s.log.Warn("failed to remove from cache", slog.String("key", cacheKey), slog.Any("err", err))
	}

	count, err := s.repo.Remove(ctx, id)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *SubscriptionService) Read(ctx context.Context, id int) (*models.Entry, error) {
	var result *models.Entry
	cacheKey := fmt.Sprintf("subscription:%d", id)
	found, err := s.cache.Get(cacheKey, &result)
	if err != nil {
		return nil, err
	}
	if found {
		return result, nil
	}
	result, err = s.repo.Read(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(cacheKey, result, time.Hour); err != nil {
		s.log.Warn("failed to add to cache", slog.String("key", cacheKey),
			slog.Any("err", err))
	}
	return result, nil
}
func (s *SubscriptionService) Update(ctx context.Context, req models.DummyEntry, id int, username string) (int, error) {
	startDate, err := time.Parse("02-01-2006", req.StartDate)
	if err != nil {
		return 0, fmt.Errorf("invalid start date: %w", err)
	}

	entry := models.Entry{
		ServiceName:   req.ServiceName,
		Username:      username,
		Price:         req.Price,
		StartDate:     startDate,
		CounterMonths: req.CounterMonths,
	}
	res, err := s.repo.Update(ctx, entry, id)
	if err != nil {
		return 0, err
	}
	s.log.Info("updated subscription in storage")

	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Set(cacheKey, entry, time.Hour); err != nil {
		s.log.Warn("failed to cache subscription", slog.String("key", cacheKey), slog.Any("err", err))
	}
	s.log.Info("updated subscription in cache")
	return res, nil
}

func (s *SubscriptionService) List(ctx context.Context, username, role string, limit, offset int) ([]*models.Entry, error) {
	var err error
	var entries []*models.Entry
	if role == "admin" {
		entries, err = s.repo.ListAll(ctx, limit, offset)
	} else {
		entries, err = s.repo.List(ctx, username, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *SubscriptionService) CountSumWithFilter(ctx context.Context, username string, req models.DummyFilterSum) (float64, error) {
	startDate, err := time.Parse("02-01-2006", req.StartDate)
	if err != nil {
		return 0, fmt.Errorf("invalid start date: %w", err)
	}

	var serviceNamePtr *string
	if req.ServiceName != "" {
		serviceNamePtr = &req.ServiceName
	}

	filter := models.FilterSum{
		Username:      username,
		ServiceName:   serviceNamePtr,
		StartDate:     startDate,
		CounterMonths: req.CounterMonths,
	}

	return s.repo.CountSum(ctx, filter)
}
