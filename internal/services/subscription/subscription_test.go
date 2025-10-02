package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type RepoMock struct{ mock.Mock }

func (m *RepoMock) CreateEntry(ctx context.Context, sub models.Entry) (int, error) {
	args := m.Called(ctx, sub)
	return args.Int(0), args.Error(1)
}
func (m *RepoMock) RemoveEntry(ctx context.Context, id int) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}
func (m *RepoMock) ReadEntry(ctx context.Context, id int) (*models.Entry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Entry), args.Error(1)
}
func (m *RepoMock) UpdateEntry(ctx context.Context, req models.Entry, id int, username string) (int, error) {
	args := m.Called(ctx, req, id, username)
	return args.Int(0), args.Error(1)
}
func (m *RepoMock) ListEntrys(ctx context.Context, username string, limit, offset int) ([]*models.Entry, error) {
	args := m.Called(ctx, username, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Entry), args.Error(1)
}
func (m *RepoMock) CountSumEntrys(ctx context.Context, filter models.FilterSum) (float64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(float64), args.Error(1)
}
func (m *RepoMock) ListAllEntrys(ctx context.Context, limit, offset int) ([]*models.Entry, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Entry), args.Error(1)
}

func (m *RepoMock) CreateEntrySubscriptionAggregator(ctx context.Context, entry models.Entry) (int, error) {
	args := m.Called(ctx, entry)
	return args.Int(0), args.Error(1)
}

func (m *RepoMock) GetSubscriptionStatus(ctx context.Context, userUID string) (bool, error) {
	args := m.Called(ctx, userUID)
	return args.Bool(0), args.Error(1)
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

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestSubscriptionService_Create(t *testing.T) {
	now := time.Now()
	entry := models.DummyEntry{
		ServiceName:   "Netflix",
		Price:         500,
		StartDate:     now.Format("02-01-2006"),
		CounterMonths: 5,
	}
	// entry для теста Create

	tests := []struct {
		name       string
		setupMocks func(r *RepoMock, c *CacheMock)
		req        models.DummyEntry
		wantID     int
		wantErr    bool
	}{
		{
			name: "success create",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				r.On("CreateEntry", mock.Anything, mock.MatchedBy(func(e models.Entry) bool {
					return e.ServiceName == entry.ServiceName &&
						e.Price == entry.Price &&
						e.CounterMonths == entry.CounterMonths
				})).Return(42, nil).Once()

				c.On("Set", "subscription:42", mock.Anything, time.Hour).Return(nil).Once()
			},
			req:     entry,
			wantID:  42,
			wantErr: false,
		},
		{
			name: "invalid date",
			setupMocks: func(_ *RepoMock, _ *CacheMock) {
			},
			req: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         500,
				StartDate:     "not-a-date",
				CounterMonths: 5,
			},
			wantID:  0,
			wantErr: true,
		},
		{
			name: "cache set error logs warning but returns id",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				r.On("CreateEntry", mock.Anything, mock.Anything).Return(7, nil).Once()
				c.On("Set", "subscription:7", mock.Anything, time.Hour).Return(errors.New("redis down")).Once()
			},
			req:     entry,
			wantID:  7,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			svc := NewSubscriptionService(repo, cache, newNoopLogger())

			tt.setupMocks(repo, cache)

			got, err := svc.CreateEntry(context.Background(), "user1", "", tt.req)
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

func TestSubscriptionService_Update(t *testing.T) {
	now := time.Now()
	entry := models.DummyEntry{
		ServiceName:   "Netflix",
		Price:         500,
		StartDate:     now.Format("02-01-2006"),
		CounterMonths: 5,
		IsActive:      true,
	}

	tests := []struct {
		name       string
		setupMocks func(r *RepoMock, c *CacheMock)
		req        models.DummyEntry
		id         int
		username   string
		wantRes    int
		wantErr    bool
	}{
		{
			name: "success update",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				r.On("UpdateEntry", mock.Anything, mock.MatchedBy(func(req models.Entry) bool {
					// Парсим дату из DummyEntry для сравнения
					startDate, _ := time.Parse("02-01-2006", entry.StartDate)
					return req.ServiceName == entry.ServiceName &&
						req.Price == entry.Price &&
						req.StartDate.Equal(startDate) &&
						req.CounterMonths == entry.CounterMonths &&
						req.IsActive == entry.IsActive
				}), 1, "user1").Return(1, nil).Once()

				c.On("Set", "subscription:1", mock.Anything, time.Hour).Return(nil).Once()
			},
			req:      entry,
			id:       1,
			username: "user1",
			wantRes:  1,
			wantErr:  false,
		},
		{
			name: "invalid date",
			setupMocks: func(_ *RepoMock, _ *CacheMock) {
			},
			req: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         500,
				StartDate:     time.Now().AddDate(0, -2, 0).Format("02-01-2006"), // 2 месяца назад
				CounterMonths: 1,                                                 // 1 месяц, чтобы endDate была месяц назад
				IsActive:      true,
			},
			id:       1,
			username: "user1",
			wantRes:  0,
			wantErr:  true,
		},
		{
			name: "cache set error logs warning but returns res",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				r.On("UpdateEntry", mock.Anything, mock.Anything, 1, "user1").Return(1, nil).Once()
				c.On("Set", "subscription:1", mock.Anything, time.Hour).Return(errors.New("redis down")).Once()
			},
			req:      entry,
			id:       1,
			username: "user1",
			wantRes:  1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			// Создаем логгер с уровнем DEBUG для отладки
			h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
			logger := slog.New(h)
			svc := NewSubscriptionService(repo, cache, logger)

			tt.setupMocks(repo, cache)

			res, err := svc.UpdateEntry(context.Background(), tt.req, tt.id, tt.username)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRes, res)
			}

			repo.AssertExpectations(t)
			cache.AssertExpectations(t)
		})
	}
}

func TestSubscriptionService_Remove(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(r *RepoMock, c *CacheMock)
		id         int
		wantCount  int
		wantErr    bool
	}{
		{
			name: "success remove",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				c.On("Invalidate", "subscription:1").Return(nil).Once()
				r.On("RemoveEntry", mock.Anything, 1).Return(1, nil).Once()
			},
			id:        1,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "cache invalidate error but proceed",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				c.On("Invalidate", "subscription:2").Return(errors.New("cache fail")).Once()
				r.On("RemoveEntry", mock.Anything, 2).Return(1, nil).Once()
			},
			id:        2,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "repo remove error",
			setupMocks: func(r *RepoMock, c *CacheMock) {
				c.On("Invalidate", "subscription:3").Return(nil).Once()
				r.On("RemoveEntry", mock.Anything, 3).Return(0, errors.New("not found")).Once()
			},
			id:        3,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			svc := NewSubscriptionService(repo, cache, newNoopLogger())

			tt.setupMocks(repo, cache)

			count, err := svc.RemoveEntry(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}

			cache.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}

func TestSubscriptionService_List(t *testing.T) {
	entries := []*models.Entry{
		{ServiceName: "Netflix", Username: "user1"},
		{ServiceName: "Spotify", Username: "user1"},
	}

	tests := []struct {
		name       string
		role       string
		username   string
		limit      int
		offset     int
		setupMocks func(r *RepoMock)
		want       []*models.Entry
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "admin role uses ListAll",
			role:     "admin",
			username: "",
			limit:    10,
			offset:   0,
			setupMocks: func(r *RepoMock) {
				r.On("ListAllEntrys", mock.Anything, 10, 0).Return(entries, nil).Once()
			},
			want:    entries,
			wantErr: false,
		},
		{
			name:     "user role uses List",
			role:     "user",
			username: "user1",
			limit:    5,
			offset:   2,
			setupMocks: func(r *RepoMock) {
				r.On("ListEntrys", mock.Anything, "user1", 5, 2).Return(entries, nil).Once()
			},
			want:    entries,
			wantErr: false,
		},
		{
			name:     "ListAll returns error",
			role:     "admin",
			username: "",
			limit:    10,
			offset:   0,
			setupMocks: func(r *RepoMock) {
				r.On("ListAllEntrys", mock.Anything, 10, 0).Return(nil, errors.New("db error")).Once()
			},
			want:    nil,
			wantErr: true,
			errMsg:  "db error",
		},
		{
			name:     "List returns error",
			role:     "user",
			username: "user1",
			limit:    10,
			offset:   0,
			setupMocks: func(r *RepoMock) {
				r.On("ListEntrys", mock.Anything, "user1", 10, 0).Return(nil, errors.New("db error")).Once()
			},
			want:    nil,
			wantErr: true,
			errMsg:  "db error",
		},
		{
			name:     "empty result for user",
			role:     "user",
			username: "user2",
			limit:    10,
			offset:   0,
			setupMocks: func(r *RepoMock) {
				r.On("ListEntrys", mock.Anything, "user2", 10, 0).Return([]*models.Entry{}, nil).Once()
			},
			want:    []*models.Entry{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			svc := NewSubscriptionService(repo, cache, newNoopLogger())

			tt.setupMocks(repo)

			got, err := svc.ListEntrys(context.Background(), tt.username, tt.role, tt.limit, tt.offset)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestSubscriptionService_Read(t *testing.T) {
	fixedTime := time.Date(2025, 8, 20, 0, 0, 0, 0, time.UTC)
	entry := &models.Entry{
		ServiceName:   "Netflix",
		Price:         500,
		Username:      "user1",
		StartDate:     fixedTime,
		CounterMonths: 5,
	}

	tests := []struct {
		name       string
		id         int
		cacheFound bool
		cacheErr   error
		repoEntry  *models.Entry
		repoErr    error
		wantEntry  *models.Entry
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "cache hit",
			id:         1,
			cacheFound: true,
			cacheErr:   nil,
			repoEntry:  nil,
			repoErr:    nil,
			wantEntry:  entry,
			wantErr:    false,
		},
		{
			name:       "cache miss then repo success",
			id:         2,
			cacheFound: false,
			cacheErr:   nil,
			repoEntry:  entry,
			repoErr:    nil,
			wantEntry:  entry,
			wantErr:    false,
		},
		{
			name:       "cache error",
			id:         3,
			cacheFound: false,
			cacheErr:   errors.New("cache unavailable"),
			repoEntry:  nil,
			repoErr:    nil,
			wantEntry:  nil,
			wantErr:    true,
			errMsg:     "cache unavailable",
		},
		{
			name:       "repo error - not found",
			id:         4,
			cacheFound: false,
			cacheErr:   nil,
			repoEntry:  nil,
			repoErr:    errors.New("not found"),
			wantEntry:  nil,
			wantErr:    true,
			errMsg:     "not found",
		},
		{
			name:       "cache miss and repo returns nil entry",
			id:         5,
			cacheFound: false,
			cacheErr:   nil,
			repoEntry:  nil,
			repoErr:    nil,
			wantEntry:  nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			svc := NewSubscriptionService(repo, cache, newNoopLogger())

			cacheKey := fmt.Sprintf("subscription:%d", tt.id)

			cache.On("Get", cacheKey, mock.Anything).Return(tt.cacheFound, tt.cacheErr).Run(func(args mock.Arguments) {
				if tt.cacheFound && tt.cacheErr == nil {
					ptrPtr := args.Get(1).(**models.Entry)
					if ptrPtr != nil {
						*ptrPtr = entry
					}
				}
			}).Once()

			if !tt.cacheFound && tt.cacheErr == nil {
				repo.On("ReadEntry", mock.Anything, tt.id).Return(tt.repoEntry, tt.repoErr).Once()

				if tt.repoEntry != nil {
					cache.On("Set", cacheKey, tt.repoEntry, time.Hour).Return(nil).Once()
				}
			}

			got, err := svc.ReadEntry(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantEntry, got)
			}

			cache.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}

func TestSubscriptionService_CountSumWithFilter(t *testing.T) {
	validDate := "01-01-2024"
	parsedDate, _ := time.Parse("02-01-2006", validDate)

	tests := []struct {
		name       string
		username   string
		req        models.DummyFilterSum
		setupMocks func(r *RepoMock)
		wantSum    float64
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "success with service name filter",
			username: "user1",
			req: models.DummyFilterSum{
				ServiceName:   "Netflix",
				StartDate:     validDate,
				CounterMonths: 5,
			},
			setupMocks: func(r *RepoMock) {
				r.On("CountSumEntrys", mock.Anything, mock.MatchedBy(func(f models.FilterSum) bool {
					return f.Username == "user1" &&
						f.ServiceName != nil && *f.ServiceName == "Netflix" &&
						f.StartDate.Equal(parsedDate) &&
						f.CounterMonths == 5
				})).Return(150.75, nil).Once()
			},
			wantSum: 150.75,
			wantErr: false,
		},
		{
			name:     "success without service name filter",
			username: "user2",
			req: models.DummyFilterSum{
				ServiceName:   "",
				StartDate:     validDate,
				CounterMonths: 3,
			},
			setupMocks: func(r *RepoMock) {
				r.On("CountSumEntrys", mock.Anything, mock.MatchedBy(func(f models.FilterSum) bool {
					return f.Username == "user2" &&
						f.ServiceName == nil &&
						f.StartDate.Equal(parsedDate) &&
						f.CounterMonths == 3
				})).Return(89.99, nil).Once()
			},
			wantSum: 89.99,
			wantErr: false,
		},
		{
			name:     "invalid start date format",
			username: "user1",
			req: models.DummyFilterSum{
				ServiceName:   "Netflix",
				StartDate:     "invalid-date",
				CounterMonths: 5,
			},
			setupMocks: func(_ *RepoMock) {},
			wantSum:    0,
			wantErr:    true,
			errMsg:     "invalid start date",
		},
		{
			name:     "repo returns error",
			username: "user1",
			req: models.DummyFilterSum{
				ServiceName:   "Netflix",
				StartDate:     validDate,
				CounterMonths: 5,
			},
			setupMocks: func(r *RepoMock) {
				r.On("CountSumEntrys", mock.Anything, mock.Anything).Return(0.0, errors.New("database error")).Once()
			},
			wantSum: 0,
			wantErr: true,
			errMsg:  "database error",
		},
		{
			name:     "zero months",
			username: "user1",
			req: models.DummyFilterSum{
				ServiceName:   "Netflix",
				StartDate:     validDate,
				CounterMonths: 0,
			},
			setupMocks: func(r *RepoMock) {
				r.On("CountSumEntrys", mock.Anything, mock.MatchedBy(func(f models.FilterSum) bool {
					return f.CounterMonths == 0
				})).Return(0.0, nil).Once()
			},
			wantSum: 0.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(RepoMock)
			cache := new(CacheMock)
			svc := NewSubscriptionService(repo, cache, newNoopLogger())

			tt.setupMocks(repo)

			got, err := svc.CountSumWithFilter(context.Background(), tt.username, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSum, got)
			}

			repo.AssertExpectations(t)
		})
	}
}
