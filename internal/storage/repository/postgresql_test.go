package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_Create(t *testing.T) {
	type args struct {
		ctx   context.Context
		entry models.Entry
	}

	tests := []struct {
		name   string
		args   args
		wantID int
	}{
		{
			name: "successful create entry",
			args: args{
				ctx: context.Background(),
				entry: models.Entry{
					ServiceName:   "Spotify",
					Price:         500,
					Username:      "testuser",
					StartDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					CounterMonths: 6,
					UserUID:       "550e8400-e29b-41d4-a716-446655440000",
				},
			},
			wantID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			// Создаем пользователя перед созданием подписки
			factory := NewTestDataFactory(storage)
			factory.CreateUser(t, "550e8400-e29b-41d4-a716-446655440000", "testuser", "test@example.com", "hashedpassword", "user")

			gotID, err := storage.CreateEntry(tt.args.ctx, tt.args.entry)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, gotID)

			// Проверяем, что подписка создана
			verification := NewTestVerification(storage)
			verification.VerifySubscriptionExists(t, gotID)
		})
	}
}

func TestStorage_Remove(t *testing.T) {
	type args struct {
		ctx context.Context
		id  int
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name             string
		args             args
		wantRowsAffected int
		wantError        bool
		setup            func(t *testing.T, factory *TestDataFactory) int
	}{
		{
			name: "successful delete entry",
			args: args{
				ctx: context.Background(),
				id:  0, // будет установлен в setup
			},
			wantRowsAffected: 1,
			wantError:        false,
			setup: func(t *testing.T, factory *TestDataFactory) int {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return factory.CreateSubscription(t, "Netflix", 1000, "testuser", startDate, 5, userUID, startDate, true)
			},
		},
		{
			name: "invalid id",
			args: args{
				ctx: context.Background(),
				id:  9999,
			},
			wantRowsAffected: 0,
			wantError:        true,
			setup: func(t *testing.T, factory *TestDataFactory) int {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreateSubscription(t, "Netflix", 1000, "testuser", startDate, 5, userUID, startDate, true)
				return 9999 // несуществующий ID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			subscriptionID := tt.setup(t, factory)
			tt.args.id = subscriptionID

			gotRowsAffected, err := storage.RemoveEntry(tt.args.ctx, tt.args.id)

			require.NoError(t, err)
			assert.Equal(t, tt.wantRowsAffected, gotRowsAffected)

			if tt.name == "successful delete entry" {
				verification := NewTestVerification(storage)
				verification.VerifySubscriptionDeleted(t, subscriptionID)
			}
		})
	}
}

func TestStorage_Read(t *testing.T) {
	type args struct {
		ctx context.Context
		id  int
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	userUID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name    string
		args    args
		want    *models.Entry
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) int
	}{
		{
			name: "successful read existing entry",
			args: args{
				ctx: context.Background(),
				id:  0, // будет установлен в setup
			},
			want: &models.Entry{
				ServiceName:   "Netflix",
				Price:         1000,
				Username:      "testuser",
				StartDate:     startDate,
				CounterMonths: 12,
				UserUID:       userUID,
			},
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) int {
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return factory.CreateSubscription(t, "Netflix", 1000, "testuser", startDate, 12, userUID, startDate, true)
			},
		},
		{
			name: "read non-existing entry",
			args: args{
				ctx: context.Background(),
				id:  999,
			},
			want:    nil,
			wantErr: true,
			setup:   func(_ *testing.T, _ *TestDataFactory) int { return 999 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			subscriptionID := tt.setup(t, factory)
			tt.args.id = subscriptionID

			got, err := storage.ReadEntry(tt.args.ctx, tt.args.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.want.ServiceName, got.ServiceName)
			assert.Equal(t, tt.want.Price, got.Price)
			assert.Equal(t, tt.want.Username, got.Username)
			assert.True(t, tt.want.StartDate.Equal(got.StartDate))
			assert.Equal(t, tt.want.CounterMonths, got.CounterMonths)
		})
	}
}

func TestStorage_CountSum(t *testing.T) {
	type args struct {
		ctx    context.Context
		filter models.FilterSum
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	netflixService := "Netflix"

	tests := []struct {
		name      string
		args      args
		wantTotal float64
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory)
	}{
		{
			name: "count sum for single subscription",
			args: args{
				ctx: context.Background(),
				filter: models.FilterSum{
					Username:      "testuser",
					ServiceName:   nil, // Нет фильтра по service_name
					StartDate:     startDate,
					CounterMonths: 12,
				},
			},
			wantTotal: 12000.0, // 1000.0 * 12 месяцев
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreateSubscription(t, "Netflix", 1000.0, "testuser", startDate, 12, userUID, startDate, true)
			},
		},
		{
			name: "count sum for filtered service",
			args: args{
				ctx: context.Background(),
				filter: models.FilterSum{
					Username:      "testuser",
					ServiceName:   &netflixService, // Фильтр по Netflix
					StartDate:     startDate,
					CounterMonths: 12,
				},
			},
			wantTotal: 12000.0, // 1000.0 * 12 месяцев (только Netflix)
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreateSubscription(t, "Netflix", 1000.0, "testuser", startDate, 12, userUID, startDate, true)
				factory.CreateSubscription(t, "Spotify", 1000.0, "testuser", startDate, 12, userUID, startDate, true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			tt.setup(t, factory)

			gotTotal, err := storage.CountSumEntrys(tt.args.ctx, tt.args.filter)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.wantTotal, gotTotal, 0.001)
		})
	}
}

func TestStorage_RegisterUser(t *testing.T) {
	type args struct {
		ctx  context.Context
		user models.User
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory)
	}{
		{
			name: "successful register user",
			args: args{
				ctx: context.Background(),
				user: models.User{
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: "hashedpassword",
					Role:         "user",
				},
			},
			wantErr: false,
			setup:   func(_ *testing.T, _ *TestDataFactory) {},
		},
		{
			name: "register user with duplicate username",
			args: args{
				ctx: context.Background(),
				user: models.User{
					Email:        "test2@example.com",
					Username:     "testuser", // duplicate username
					PasswordHash: "hashedpassword2",
					Role:         "user",
				},
			},
			wantErr: true,
			setup: func(t *testing.T, factory *TestDataFactory) {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			tt.setup(t, factory)

			var uid string
			err := storage.DB.QueryRowContext(tt.args.ctx,
				`INSERT INTO users (email, username, password_hash, role) 
				 VALUES ($1, $2, $3, $4) RETURNING uid`,
				tt.args.user.Email,
				tt.args.user.Username,
				tt.args.user.PasswordHash,
				tt.args.user.Role).Scan(&uid)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, uid)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, uid)

			// Проверяем, что пользователь создан
			verification := NewTestVerification(storage)
			verification.VerifyUserExists(t, uid)
		})
	}
}

func TestStorage_GetUserByUsername(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
	}

	tests := []struct {
		name    string
		args    args
		want    *models.User
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful get user by username",
			args: args{
				ctx:      context.Background(),
				username: "testuser",
			},
			want: &models.User{
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashedpassword",
				Role:         "user",
			},
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return userUID
			},
		},
		{
			name: "get non-existing user",
			args: args{
				ctx:      context.Background(),
				username: "nonexistent",
			},
			want:    nil,
			wantErr: true,
			setup:   func(_ *testing.T, _ *TestDataFactory) string { return "" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			if tt.want != nil {
				tt.want.UUID = userUID
			}

			got, err := storage.GetUserByUsername(tt.args.ctx, tt.args.username)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Username, got.Username)
			assert.Equal(t, tt.want.PasswordHash, got.PasswordHash)
			assert.Equal(t, tt.want.Role, got.Role)
		})
	}
}

// TestStorage_UpdateEntry удален, так как метод UpdateEntry изменил сигнатуру
func TestStorage_UpdateEntry_DISABLED(t *testing.T) {
	type args struct {
		ctx   context.Context
		entry models.Entry
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	userUID := uuid.New().String()

	tests := []struct {
		name             string
		args             args
		wantRowsAffected int
		wantErr          bool
		setup            func(s *Storage) int
		verify           func(t *testing.T, s *Storage, id int)
	}{
		{
			name: "successful update entry",
			args: args{
				ctx: context.Background(),
				entry: models.Entry{
					ID:              1, // будет установлен в setup
					ServiceName:     "Netflix Updated",
					Price:           1500,
					Username:        "testuser",
					StartDate:       startDate,
					CounterMonths:   24,
					UserUID:         userUID,
					NextPaymentDate: startDate.AddDate(0, 1, 0),
					IsActive:        true,
				},
			},
			wantRowsAffected: 1,
			wantErr:          false,
			setup: func(s *Storage) int {
				// Создаем пользователя
				_, err := s.DB.Exec(`INSERT INTO users (uid, username, email, password_hash, role) 
					VALUES ($1, $2, $3, $4, $5)`,
					userUID, "testuser", "test@example.com", "hashedpassword", "user")
				require.NoError(t, err)

				// Создаем подписку
				var id int
				err = s.DB.QueryRow(`INSERT INTO subscriptions 
					(service_name, price, username, start_date, counter_months, user_uid, next_payment_date, is_active)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
					"Netflix", 1000, "testuser", startDate, 12, userUID, startDate, true).Scan(&id)
				require.NoError(t, err)
				return id
			},
			verify: func(t *testing.T, s *Storage, id int) {
				var serviceName string
				var price float64
				var counterMonths int
				err := s.DB.QueryRow("SELECT service_name, price, counter_months FROM subscriptions WHERE id = $1", id).
					Scan(&serviceName, &price, &counterMonths)
				require.NoError(t, err)
				assert.Equal(t, "Netflix Updated", serviceName)
				assert.Equal(t, 1500.0, price)
				assert.Equal(t, 24, counterMonths)
			},
		},
		{
			name: "update non-existing entry",
			args: args{
				ctx: context.Background(),
				entry: models.Entry{
					ID:              999,
					ServiceName:     "Netflix",
					Price:           1000,
					Username:        "testuser",
					StartDate:       startDate,
					CounterMonths:   12,
					UserUID:         userUID,
					NextPaymentDate: startDate,
					IsActive:        true,
				},
			},
			wantRowsAffected: 0,
			wantErr:          false,
			setup:            func(_ *Storage) int { return 999 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			entryID := tt.setup(storage)
			tt.args.entry.ID = entryID

			gotRowsAffected, err := storage.UpdateEntry(tt.args.ctx, tt.args.entry, tt.args.entry.ID, tt.args.entry.Username)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantRowsAffected, gotRowsAffected)
			if tt.verify != nil {
				tt.verify(t, storage, entryID)
			}
		})
	}
}

func TestCheckDatabaseReady(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, storage *Storage)
		wantError    bool
		errorContain string
	}{
		{
			name: "table exists",
			setup: func(_ *testing.T, _ *Storage) {
				// Таблица уже создается в setupTestDatabase
			},
			wantError: false,
		},
		{
			name: "table missing",
			setup: func(t *testing.T, storage *Storage) {
				// Удаляем таблицы в правильном порядке, учитывая foreign key constraints
				_, err := storage.DB.Exec(`DROP TABLE IF EXISTS yookassa_payments CASCADE`)
				require.NoError(t, err)
				_, err = storage.DB.Exec(`DROP TABLE IF EXISTS yookassa_payment_tokens CASCADE`)
				require.NoError(t, err)
				_, err = storage.DB.Exec(`DROP TABLE IF EXISTS subscriptions CASCADE`)
				require.NoError(t, err)
				_, err = storage.DB.Exec(`DROP TABLE IF EXISTS users CASCADE`)
				require.NoError(t, err)
			},
			wantError:    true,
			errorContain: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()
			tt.setup(t, storage)

			err := CheckDatabaseReady(storage)
			if tt.wantError {
				require.Error(t, err)
				if tt.errorContain != "" {
					assert.Contains(t, err.Error(), tt.errorContain)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStorage_FindSubscriptionExpiringTomorrow(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, factory *TestDataFactory) error
		wantCount int
		wantError bool
	}{
		{
			name: "one subscription expires tomorrow",
			setup: func(t *testing.T, factory *TestDataFactory) error {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "somehash", "user")

				// Создаем подписку, которая истекает завтра
				_, err := factory.storage.DB.Exec(`
					INSERT INTO subscriptions 
						(service_name, price, username, start_date, counter_months, user_uid, next_payment_date, is_active) 
					VALUES 
						('TestService', 100, 'testuser', CURRENT_DATE - INTERVAL '1 month' + INTERVAL '1 day', 1, $1, CURRENT_DATE + INTERVAL '1 day', true)
				`, userUID)
				return err
			},
			wantCount: 1,
			wantError: false,
		},
		{
			name:      "no subscriptions expire tomorrow",
			setup:     func(_ *testing.T, _ *TestDataFactory) error { return nil },
			wantCount: 0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			err := tt.setup(t, factory)
			require.NoError(t, err)

			ctx := context.Background()
			res, err := storage.FindSubscriptionExpiringTomorrow(ctx)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, res, tt.wantCount)
			}
		})
	}
}
