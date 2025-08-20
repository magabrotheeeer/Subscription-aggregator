package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type RepoMock struct{ mock.Mock }

func (m *RepoMock) Create(ctx context.Context, sub models.Entry) (int, error) {
	args := m.Called(ctx, sub)
	return args.Int(0), args.Error(1)
}

func (m *RepoMock) Remove(ctx context.Context, id int) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

func (m *RepoMock) Read(ctx context.Context, id int) (*models.Entry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Entry), args.Error(1)
}

func (m *RepoMock) Update(ctx context.Context, entry models.Entry, id int) (int, error) {
	args := m.Called(ctx, entry, id)
	return args.Int(0), args.Error(1)
}

func (m *RepoMock) List(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error) {
	args := m.Called(ctx, username, limit, offset)
	return args.Get(0).([]*models.Entry), args.Error(1)
}

func (m *RepoMock) CountSum(ctx context.Context, filter models.FilterSum) (float64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(float64), args.Error(1)
}

func (m *RepoMock) ListAll(ctx context.Context, limit, offset int) ([]*models.Entry, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.Entry), args.Error(1)
}

type CacheMock struct{ mock.Mock }

func (m *CacheMock) Get(key string, result any) (bool, error) {
	args := m.Called(key, result)
	return args.Bool(0), args.Error(1)
}

func (m *CacheMock) Set(key string, value any, expiration time.Duration) error {
	return m.Called(key, value, expiration).Error(0)
}

func (m *CacheMock) Invalidate(key string) error {
	return m.Called(key).Error(0)
}

func NewNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestSusbcription_Create(t *testing.T) {
	format := "02-01-2006"
	now := "01-01-2024"
	timeNow, _ := time.Parse(format, now)
	dummyReq := models.DummyEntry{
		ServiceName:   "Netflix",
		Price:         500,
		StartDate:     now,
		CounterMonths: 5,
	}
	entry := models.Entry{
		ServiceName:   dummyReq.ServiceName,
		Price:         dummyReq.Price,
		Username:      "testuser",
		StartDate:     timeNow,
		CounterMonths: 5,
	}

	tests := []struct {
		name       string
		setupMocks func(repo *RepoMock, cache *CacheMock)
		req        models.DummyEntry
		wantID     int
		wantErr    bool
	}{
		{
			name: "success create",
			setupMocks: func(repo *RepoMock, cache *CacheMock) {
				repo.On("Create", mock.Anything, entry).Return(42, nil).Once()
				cache.On("Set", "subscription:42", entry, time.Hour).Return(nil).Once()
			},
			req:     dummyReq,
			wantID:  42,
			wantErr: false,
		},
		{
			name: "invalid date",
			setupMocks: func(repo *RepoMock, cache *CacheMock) {

			},
			req: models.DummyEntry{
				ServiceName:   dummyReq.ServiceName,
				Price:         dummyReq.Price,
				StartDate:     "not a date",
				CounterMonths: dummyReq.CounterMonths,
			},
			wantID:  0,
			wantErr: true,
		},
		{
			name: "cache error logs warning but returns id",
			setupMocks: func(repo *RepoMock, cache *CacheMock) {
				repo.On("Create", mock.Anything, entry).Return(7, nil).Once()
				cache.On("Set", "subscription:7", entry, time.Hour).Return(errors.New("redis down")).Once()
			},
			req:     dummyReq,
			wantID:  7,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			svc := NewSubscriptionService(repo, cache, NewNoopLogger())

			tt.setupMocks(repo, cache)

			got, err := svc.Create(context.Background(), "testuser", tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, got)
			}

			repo.AssertExpectations(t)
			cache.AssertExpectations(t)
		})
	}
}
