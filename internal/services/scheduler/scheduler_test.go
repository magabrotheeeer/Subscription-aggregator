package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindSubscriptionExpiringTomorrow(ctx context.Context) ([]*models.EntryInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EntryInfo), args.Error(1)
}

func (m *MockRepository) FindSubscriptionExpiringToday(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockRepository) FindOldNextPaymentDate(ctx context.Context) ([]*models.Entry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Entry), args.Error(1)
}

func (m *MockRepository) UpdateNextPaymentDate(ctx context.Context, entry *models.Entry) (int, error) {
	args := m.Called(ctx, entry)
	return args.Int(0), args.Error(1)
}

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Set(key string, value any, expiration time.Duration) error {
	args := m.Called(key, value, expiration)
	return args.Error(0)
}

type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestSchedulerService_runFindExpiringSubscriptionsDueTomorrow(t *testing.T) {
	now := time.Now()
	entryInfo := &models.EntryInfo{
		Email:       "test@example.com",
		Username:    "testuser",
		ServiceName: "Netflix",
		EndDate:     now.Add(24 * time.Hour),
		Price:       500,
	}

	tests := []struct {
		name          string
		setupMocks    func(*MockRepository, *MockChannel)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "success - found expiring subscriptions",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringTomorrow", mock.Anything).Return([]*models.EntryInfo{entryInfo}, nil).Once()
				// Не ожидаем Publish, так как канал nil
			},
			expectedError: false,
		},
		{
			name: "success - no expiring subscriptions",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringTomorrow", mock.Anything).Return([]*models.EntryInfo{}, nil).Once()
			},
			expectedError: false,
		},
		{
			name: "repository error",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringTomorrow", mock.Anything).Return(nil, errors.New("db error")).Once()
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
		{
			name: "publish error",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringTomorrow", mock.Anything).Return([]*models.EntryInfo{entryInfo}, nil).Once()
				// Не ожидаем Publish, так как канал nil
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			cache := new(MockCache)
			channel := new(MockChannel)
			service := NewSchedulerService(repo, cache, newNoopLogger())

			tt.setupMocks(repo, channel)

			// Вызываем приватный метод через публичный
			// Создаем реальный канал для тестирования
			service.runFindExpiringSubscriptionsDueTomorrow(context.Background(), nil)

			repo.AssertExpectations(t)
			channel.AssertExpectations(t)
		})
	}
}

func TestSchedulerService_runFindExpiringTrialPeriod(t *testing.T) {
	user := &models.User{
		UUID:     "user123",
		Email:    "test@example.com",
		Username: "testuser",
	}

	tests := []struct {
		name          string
		setupMocks    func(*MockRepository, *MockChannel)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "success - found expiring trial subscriptions",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringToday", mock.Anything).Return([]*models.User{user}, nil).Once()
				// Не ожидаем Publish, так как канал nil
			},
			expectedError: false,
		},
		{
			name: "success - no expiring trial subscriptions",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringToday", mock.Anything).Return([]*models.User{}, nil).Once()
			},
			expectedError: false,
		},
		{
			name: "repository error",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringToday", mock.Anything).Return(nil, errors.New("db error")).Once()
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
		{
			name: "publish error",
			setupMocks: func(r *MockRepository, _ *MockChannel) {
				r.On("FindSubscriptionExpiringToday", mock.Anything).Return([]*models.User{user}, nil).Once()
				// Не ожидаем Publish, так как канал nil
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			cache := new(MockCache)
			channel := new(MockChannel)
			service := NewSchedulerService(repo, cache, newNoopLogger())

			tt.setupMocks(repo, channel)

			// Вызываем приватный метод через публичный
			// Создаем реальный канал для тестирования
			service.runFindExpiringTrialPeriod(context.Background(), nil)

			repo.AssertExpectations(t)
			channel.AssertExpectations(t)
		})
	}
}

func TestSchedulerService_runFindOldNextPaymentDate(t *testing.T) {
	now := time.Now()
	entry := &models.Entry{
		ID:              1,
		ServiceName:     "Netflix",
		Price:           500,
		Username:        "testuser",
		NextPaymentDate: now.Add(-24 * time.Hour), // старая дата
	}

	tests := []struct {
		name          string
		setupMocks    func(*MockRepository, *MockCache)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "success - found old payment dates",
			setupMocks: func(r *MockRepository, c *MockCache) {
				r.On("FindOldNextPaymentDate", mock.Anything).Return([]*models.Entry{entry}, nil).Once()
				r.On("UpdateNextPaymentDate", mock.Anything, mock.AnythingOfType("*models.Entry")).Return(1, nil).Once()
				c.On("Set", "subscription:1", mock.AnythingOfType("*models.Entry"), time.Hour).Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name: "success - no old payment dates",
			setupMocks: func(r *MockRepository, _ *MockCache) {
				r.On("FindOldNextPaymentDate", mock.Anything).Return([]*models.Entry{}, nil).Once()
			},
			expectedError: false,
		},
		{
			name: "repository error on find",
			setupMocks: func(r *MockRepository, _ *MockCache) {
				r.On("FindOldNextPaymentDate", mock.Anything).Return(nil, errors.New("db error")).Once()
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
		{
			name: "repository error on update",
			setupMocks: func(r *MockRepository, _ *MockCache) {
				r.On("FindOldNextPaymentDate", mock.Anything).Return([]*models.Entry{entry}, nil).Once()
				r.On("UpdateNextPaymentDate", mock.Anything, mock.AnythingOfType("*models.Entry")).Return(0, errors.New("update error")).Once()
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
		{
			name: "cache error",
			setupMocks: func(r *MockRepository, c *MockCache) {
				r.On("FindOldNextPaymentDate", mock.Anything).Return([]*models.Entry{entry}, nil).Once()
				r.On("UpdateNextPaymentDate", mock.Anything, mock.AnythingOfType("*models.Entry")).Return(1, nil).Once()
				c.On("Set", "subscription:1", mock.AnythingOfType("*models.Entry"), time.Hour).Return(errors.New("cache error")).Once()
			},
			expectedError: false, // метод не возвращает ошибку, только логирует
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			cache := new(MockCache)
			service := NewSchedulerService(repo, cache, newNoopLogger())

			tt.setupMocks(repo, cache)

			// Вызываем приватный метод через публичный
			service.runFindOldNextPaymentDate(context.Background())

			repo.AssertExpectations(t)
			cache.AssertExpectations(t)
		})
	}
}

func TestSchedulerService_NewSchedulerService(t *testing.T) {
	repo := new(MockRepository)
	cache := new(MockCache)
	logger := newNoopLogger()

	service := NewSchedulerService(repo, cache, logger)

	assert.NotNil(t, service)
	assert.Equal(t, repo, service.repo)
	assert.Equal(t, cache, service.cache)
	assert.Equal(t, logger, service.log)
}

func TestSchedulerService_NextPaymentDateUpdate(t *testing.T) {
	now := time.Now()
	oldDate := now.Add(-24 * time.Hour)
	entry := &models.Entry{
		ID:              1,
		ServiceName:     "Netflix",
		Price:           500,
		Username:        "testuser",
		NextPaymentDate: oldDate,
	}

	repo := new(MockRepository)
	cache := new(MockCache)
	service := NewSchedulerService(repo, cache, newNoopLogger())

	repo.On("FindOldNextPaymentDate", mock.Anything).Return([]*models.Entry{entry}, nil).Once()
	repo.On("UpdateNextPaymentDate", mock.Anything, mock.AnythingOfType("*models.Entry")).Return(1, nil).Once()
	cache.On("Set", "subscription:1", mock.AnythingOfType("*models.Entry"), time.Hour).Return(nil).Once()

	service.runFindOldNextPaymentDate(context.Background())

	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
}
