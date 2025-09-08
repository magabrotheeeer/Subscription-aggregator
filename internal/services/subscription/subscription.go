// Package services содержит бизнес-логику для управления подписками и кешированием.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// SubscriptionRepository определяет методы для работы с подписками в хранилище.
type SubscriptionRepository interface {
	// Create добавляет новую подписку и возвращает её ID.
	CreateEntry(ctx context.Context, sub models.Entry) (int, error)
	// Remove удаляет подписку по ID и возвращает количество удалённых записей.
	RemoveEntry(ctx context.Context, id int) (int, error)
	// Read возвращает подписку по ID.
	ReadEntry(ctx context.Context, id int) (*models.Entry, error)
	// Update обновляет данные подписки по ID.
	UpdateEntry(ctx context.Context, entry models.Entry) (int, error)
	// List возвращает список подписок для пользователя с пагинацией.
	ListEntrys(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error)
	// CountSum подсчитывает сумму по фильтру.
	CountSumEntrys(ctx context.Context, entry models.FilterSum) (float64, error)
	// ListAll возвращает список всех подписок с пагинацией.
	ListAllEntrys(ctx context.Context, limit, offset int) ([]*models.Entry, error)
}

// Cache описывает методы для кэширования данных.
type Cache interface {
	// Get пытается получить значение из кеша по ключу.
	Get(key string, result any) (bool, error)
	// Set сохраняет значение в кеш с временем жизни.
	Set(key string, value any, expiration time.Duration) error
	// Invalidate удаляет значение из кеша по ключу.
	Invalidate(key string) error
}

// SubscriptionService реализует бизнес-логику работы с подписками, включая кеширование.
type SubscriptionService struct {
	repo  SubscriptionRepository
	cache Cache
	log   *slog.Logger
}

// NewSubscriptionService создает новый экземпляр SubscriptionService.
func NewSubscriptionService(repo SubscriptionRepository, cache Cache, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

// Create создает новую подписку для пользователя, кеширует её и возвращает ID.
func (s *SubscriptionService) CreateEntry(ctx context.Context, userName string, userUID string, req models.DummyEntry) (int, error) {
	startDate, err := time.Parse("02-01-2006", req.StartDate)
	if err != nil {
		return 0, fmt.Errorf("invalid start date: %w", err)
	}
	endDate := startDate.AddDate(0, req.CounterMonths, 0)
	today := time.Now().Truncate(24 * time.Hour)
	if endDate.Before(today) {
		return 0, fmt.Errorf("subscription end date must not be earlier than today")
	}

	nextPaymentDate := startDate.AddDate(0, 1, 0)
	entry := models.Entry{
		ServiceName:     req.ServiceName,
		Username:        userName,
		Price:           req.Price,
		StartDate:       startDate,
		CounterMonths:   req.CounterMonths,
		NextPaymentDate: nextPaymentDate,
		IsActive:        true,
		UserUID:         userUID,
	}

	id, err := s.repo.CreateEntry(ctx, entry)
	if err != nil {
		return 0, err
	}

	s.log.Info("created new subscription", slog.Int("id", id))

	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Set(cacheKey, entry, time.Hour); err != nil {
		s.log.Warn("failed to cache subscription", slog.String("key", cacheKey), sl.Err(err))
	}
	s.log.Info("created new subscription in cache")

	return id, nil
}

// Remove удаляет подписку по ID и инвалидирует кеш.
func (s *SubscriptionService) RemoveEntry(ctx context.Context, id int) (int, error) {
	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Invalidate(cacheKey); err != nil {
		s.log.Warn("failed to remove from cache", slog.String("key", cacheKey), sl.Err(err))
	}

	count, err := s.repo.RemoveEntry(ctx, id)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Read возвращает подписку по ID, используя кеш или репозиторий.
func (s *SubscriptionService) ReadEntry(ctx context.Context, id int) (*models.Entry, error) {
	var result *models.Entry
	cacheKey := fmt.Sprintf("subscription:%d", id)
	found, err := s.cache.Get(cacheKey, &result)
	if err != nil {
		return nil, err
	}
	if found {
		return result, nil
	}
	result, err = s.repo.ReadEntry(ctx, id)
	if err != nil {
		return nil, err
	}

	if result != nil {
		if err := s.cache.Set(cacheKey, result, time.Hour); err != nil {
			s.log.Warn("failed to add to cache", slog.String("key", cacheKey), sl.Err(err))
		}
	}
	return result, nil
}

// Update обновляет подписку и обновляет кеш.
func (s *SubscriptionService) UpdateEntry(ctx context.Context, req models.DummyEntry, id int, username string) (int, error) {
	startDate, err := time.Parse("02-01-2006", req.StartDate)
	if err != nil {
		return 0, fmt.Errorf("invalid start date: %w", err)
	}
	endDate := startDate.AddDate(0, req.CounterMonths, 0)
	today := time.Now().Truncate(24 * time.Hour)
	if endDate.Before(today) {
		return 0, fmt.Errorf("subscription end date must not be earlier than today")
	}

	entry := models.Entry{
		ServiceName:   req.ServiceName,
		Username:      username,
		Price:         req.Price,
		StartDate:     startDate,
		CounterMonths: req.CounterMonths,
		IsActive:      req.IsActive,
		ID:            id,
	}
	res, err := s.repo.UpdateEntry(ctx, entry)
	if err != nil {
		return 0, err
	}
	s.log.Info("updated subscription in storage")

	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Set(cacheKey, entry, time.Hour); err != nil {
		s.log.Warn("failed to cache subscription", slog.String("key", cacheKey), sl.Err(err))
	}
	s.log.Info("updated subscription in cache")
	return res, nil
}

// List возвращает список подписок в зависимости от роли пользователя.
func (s *SubscriptionService) ListEntrys(ctx context.Context, username, role string, limit, offset int) ([]*models.Entry, error) {
	var err error
	var entries []*models.Entry
	if role == "admin" {
		entries, err = s.repo.ListAllEntrys(ctx, limit, offset)
	} else {
		entries, err = s.repo.ListEntrys(ctx, username, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// CountSumWithFilter считает сумму подписок по заданным фильтрам.
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

	return s.repo.CountSumEntrys(ctx, filter)
}

func (s *SubscriptionService) CreateEntrySubscriptionAggregator(ctx context.Context, username, userUID string) (int, error) {
	entry := models.Entry{
		ServiceName:     "Subscription-Aggregator",
		Price:           0,
		IsActive:        true,
		CounterMonths:   1,
		Username:        username,
		UserUID:         userUID,
		StartDate:       time.Now(),
		NextPaymentDate: time.Now().AddDate(0, 1, 0),
	}
	id, err := s.repo.CreateEntry(ctx, entry)
	if err != nil {
		return 0, err
	}

	s.log.Info("created new subscription", slog.Int("id", id))

	cacheKey := fmt.Sprintf("subscription:%d", id)
	if err := s.cache.Set(cacheKey, entry, time.Hour); err != nil {
		s.log.Warn("failed to cache subscription", slog.String("key", cacheKey), sl.Err(err))
	}
	s.log.Info("created new subscription in cache")
	return id, nil
}

