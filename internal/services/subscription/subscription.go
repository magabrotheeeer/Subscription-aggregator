// Package services содержит бизнес-логику для управления подписками и кешированием.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// SubscriptionRepository определяет методы для работы с подписками в хранилище.
type SubscriptionRepository interface {
	// Create добавляет новую подписку и возвращает её ID.
	Create(ctx context.Context, sub models.Entry) (int, error)
	// Remove удаляет подписку по ID и возвращает количество удалённых записей.
	Remove(ctx context.Context, id int) (int, error)
	// Read возвращает подписку по ID.
	Read(ctx context.Context, id int) (*models.Entry, error)
	// Update обновляет данные подписки по ID.
	Update(ctx context.Context, entry models.Entry, id int) (int, error)
	// List возвращает список подписок для пользователя с пагинацией.
	List(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error)
	// CountSum подсчитывает сумму по фильтру.
	CountSum(ctx context.Context, entry models.FilterSum) (float64, error)
	// ListAll возвращает список всех подписок с пагинацией.
	ListAll(ctx context.Context, limit, offset int) ([]*models.Entry, error)
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
func (s *SubscriptionService) Create(ctx context.Context, userName string, req models.DummyEntry) (int, error) {
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
	s.log.Info("created new subscription in cache")

	return id, nil
}

// Remove удаляет подписку по ID и инвалидирует кеш.
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

// Read возвращает подписку по ID, используя кеш или репозиторий.
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

	if result != nil {
		if err := s.cache.Set(cacheKey, result, time.Hour); err != nil {
			s.log.Warn("failed to add to cache", slog.String("key", cacheKey),
				slog.Any("err", err))
		}
	}
	return result, nil
}

// Update обновляет подписку и обновляет кеш.
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

// List возвращает список подписок в зависимости от роли пользователя.
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

	return s.repo.CountSum(ctx, filter)
}
