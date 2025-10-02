package repository

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_ListEntrys(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
		limit    int
		offset   int
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		args      args
		wantCount int
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory)
	}{
		{
			name: "successful list entries with pagination",
			args: args{
				ctx:      context.Background(),
				username: "testuser",
				limit:    10,
				offset:   0,
			},
			wantCount: 2,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreateSubscription(t, "Netflix", 1000.0, "testuser", startDate, 12, userUID, startDate, true)
				factory.CreateSubscription(t, "Spotify", 500.0, "testuser", startDate, 6, userUID, startDate, true)
			},
		},
		{
			name: "list entries for non-existing user",
			args: args{
				ctx:      context.Background(),
				username: "nonexistent",
				limit:    10,
				offset:   0,
			},
			wantCount: 0,
			wantErr:   false,
			setup:     func(_ *testing.T, _ *TestDataFactory) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			tt.setup(t, factory)

			got, err := storage.ListEntrys(tt.args.ctx, tt.args.username, tt.args.limit, tt.args.offset)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCount)
			}
		})
	}
}

func TestStorage_ListAllEntrys(t *testing.T) {
	type args struct {
		ctx    context.Context
		limit  int
		offset int
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		args      args
		wantCount int
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory)
	}{
		{
			name: "successful list all entries",
			args: args{
				ctx:    context.Background(),
				limit:  10,
				offset: 0,
			},
			wantCount: 3,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) {
				userUID1 := uuid.New().String()
				userUID2 := uuid.New().String()

				// Создаем пользователей
				factory.CreateUser(t, userUID1, "user1", "user1@example.com", "hashedpassword1", "user")
				factory.CreateUser(t, userUID2, "user2", "user2@example.com", "hashedpassword2", "user")

				// Создаем подписки для разных пользователей
				factory.CreateSubscription(t, "Netflix", 1000.0, "user1", startDate, 12, userUID1, startDate, true)
				factory.CreateSubscription(t, "Spotify", 500.0, "user1", startDate, 6, userUID1, startDate, true)
				factory.CreateSubscription(t, "Disney+", 800.0, "user2", startDate, 12, userUID2, startDate, true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			tt.setup(t, factory)

			got, err := storage.ListAllEntrys(tt.args.ctx, tt.args.limit, tt.args.offset)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCount)
			}
		})
	}
}

func TestStorage_GetUser(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
	}

	trialEndDate := time.Now().AddDate(0, 0, 7)       // 7 дней от сегодня
	subscriptionExpiry := time.Now().AddDate(0, 1, 0) // 1 месяц от сегодня

	tests := []struct {
		name    string
		args    args
		want    *models.User
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful get user by UID",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
			},
			want: &models.User{
				Email:              "test@example.com",
				Username:           "testuser",
				PasswordHash:       "hashedpassword",
				Role:               "user",
				TrialEndDate:       &trialEndDate,
				SubscriptionStatus: "trial",
				SubscriptionExpire: &subscriptionExpiry,
			},
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUserWithSubscription(t, userUID, "testuser", "test@example.com", "hashedpassword", "user",
					trialEndDate, subscriptionExpiry, "trial")
				return userUID
			},
		},
		{
			name: "get non-existing user by UID",
			args: args{
				ctx:     context.Background(),
				userUID: "non-existing-uid",
			},
			want:    nil,
			wantErr: true,
			setup:   func(_ *testing.T, _ *TestDataFactory) string { return "non-existing-uid" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID
			if tt.want != nil {
				tt.want.UUID = userUID
			}

			got, err := storage.GetUser(tt.args.ctx, tt.args.userUID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.want.UUID, got.UUID)
				assert.Equal(t, tt.want.Email, got.Email)
				assert.Equal(t, tt.want.Username, got.Username)
				assert.Equal(t, tt.want.PasswordHash, got.PasswordHash)
				assert.Equal(t, tt.want.Role, got.Role)
				assert.Equal(t, tt.want.SubscriptionStatus, got.SubscriptionStatus)
			}
		})
	}
}

func TestStorage_FindPaymentToken(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
		token   string
	}

	tests := []struct {
		name      string
		args      args
		wantID    int
		wantFound bool
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful find existing payment token",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
				token:   "token123",
			},
			wantID:    1,
			wantFound: true,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreatePaymentToken(t, userUID, "token123")
				return userUID
			},
		},
		{
			name: "find non-existing payment token",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
				token:   "nonexistent",
			},
			wantID:    0,
			wantFound: false,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			gotID, gotFound, err := storage.FindPaymentToken(tt.args.ctx, tt.args.userUID, tt.args.token)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}

func TestStorage_CreatePaymentToken(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
		token   string
	}

	tests := []struct {
		name    string
		args    args
		wantID  int
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful create payment token",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
				token:   "new_token_123",
			},
			wantID:  1,
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			gotID, err := storage.CreatePaymentToken(tt.args.ctx, tt.args.userUID, tt.args.token)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}
		})
	}
}

func TestStorage_ListPaymentTokens(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
	}

	tests := []struct {
		name      string
		args      args
		wantCount int
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful list payment tokens",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
			},
			wantCount: 2,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreatePaymentToken(t, userUID, "token1")
				factory.CreatePaymentToken(t, userUID, "token2")
				return userUID
			},
		},
		{
			name: "list tokens for user with no tokens",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
			},
			wantCount: 0,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			got, err := storage.ListPaymentTokens(tt.args.ctx, tt.args.userUID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCount)
			}
		})
	}
}

func TestStorage_SavePayment(t *testing.T) {
	type args struct {
		ctx     context.Context
		payload *paymentwebhook.Payload
	}

	tests := []struct {
		name    string
		args    args
		wantID  int
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful save payment",
			args: args{
				ctx: context.Background(),
				payload: &paymentwebhook.Payload{
					Object: struct {
						ID     string `json:"id"`
						Status string `json:"status"`
						Amount struct {
							Value    string `json:"value"`
							Currency string `json:"currency"`
						} `json:"amount"`
						PaymentMethod struct {
							ID string `json:"id"`
						} `json:"payment_method"`
						Metadata map[string]string `json:"metadata"`
					}{
						ID:     "payment_123",
						Status: "succeeded",
						Amount: struct {
							Value    string `json:"value"`
							Currency string `json:"currency"`
						}{
							Value:    "100.00",
							Currency: "RUB",
						},
						PaymentMethod: struct {
							ID string `json:"id"`
						}{
							ID: "card_123",
						},
						Metadata: map[string]string{
							"user_uid":        "", // будет установлен в setup
							"subscription_id": "sub_123",
						},
					},
				},
			},
			wantID:  1,
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				return userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.payload.Object.Metadata["user_uid"] = userUID
			amount, _ := strconv.ParseFloat(tt.args.payload.Object.Amount.Value, 64)

			gotID, err := storage.SavePayment(tt.args.ctx, tt.args.payload, int64(amount), userUID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}
		})
	}
}

func TestStorage_GetActiveSubscriptionIDByUserUID(t *testing.T) {
	type args struct {
		ctx         context.Context
		userUID     string
		serviceName string
	}

	tests := []struct {
		name    string
		args    args
		wantID  string
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful get active subscription ID",
			args: args{
				ctx:         context.Background(),
				userUID:     "", // будет установлен в setup
				serviceName: "Subscription-Aggregator",
			},
			wantID:  "1",
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")
				factory.CreateSubscription(t, "Subscription-Aggregator", 200.0, "testuser", time.Now(), 1, userUID, time.Now(), true)
				return userUID
			},
		},
		{
			name: "get non-existing subscription",
			args: args{
				ctx:         context.Background(),
				userUID:     "", // будет установлен в setup
				serviceName: "Non-existing-service",
			},
			wantID:  "",
			wantErr: true,
			setup:   func(_ *testing.T, _ *TestDataFactory) string { return uuid.New().String() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			gotID, err := storage.GetActiveSubscriptionIDByUserUID(tt.args.ctx, tt.args.userUID, tt.args.serviceName)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}
		})
	}
}

func TestStorage_GetSubscriptionStatus(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
	}

	tests := []struct {
		name       string
		args       args
		wantStatus string
		wantErr    bool
		setup      func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful get subscription status",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
			},
			wantStatus: "active",
			wantErr:    false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				// Создаем пользователя с активной подпиской
				_, err := factory.storage.DB.Exec(`INSERT INTO users (uid, username, email, password_hash, role, subscription_status) 
					VALUES ($1, $2, $3, $4, $5, $6)`,
					userUID, "testuser", "test@example.com", "hashedpassword", "user", "active")
				require.NoError(t, err)
				return userUID
			},
		},
		{
			name: "get status for non-existing user",
			args: args{
				ctx:     context.Background(),
				userUID: "non-existing-uid",
			},
			wantStatus: "",
			wantErr:    true,
			setup:      func(_ *testing.T, _ *TestDataFactory) string { return "non-existing-uid" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			gotStatus, err := storage.GetSubscriptionStatus(tt.args.ctx, tt.args.userUID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, gotStatus)
			}
		})
	}
}

func TestStorage_UpdateStatusActiveForSubscription(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
		status  string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful update subscription status to active",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
				status:  "active",
			},
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				// Создаем пользователя с trial статусом
				_, err := factory.storage.DB.Exec(`INSERT INTO users (uid, username, email, password_hash, role, subscription_status, subscription_expiry) 
					VALUES ($1, $2, $3, $4, $5, $6, $7)`,
					userUID, "testuser", "test@example.com", "hashedpassword", "user", "trial", time.Now().AddDate(0, 1, 0))
				require.NoError(t, err)
				return userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			err := storage.UpdateStatusActiveForSubscription(tt.args.ctx, tt.args.userUID, tt.args.status)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Проверяем, что статус обновился
				verification := NewTestVerification(storage)
				verification.VerifyUserSubscriptionStatus(t, userUID, "active")
			}
		})
	}
}

func TestStorage_UpdateStatusCancelForSubscription(t *testing.T) {
	type args struct {
		ctx     context.Context
		userUID string
		status  string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		setup   func(t *testing.T, factory *TestDataFactory) string
	}{
		{
			name: "successful update subscription status to expired",
			args: args{
				ctx:     context.Background(),
				userUID: "", // будет установлен в setup
				status:  "expired",
			},
			wantErr: false,
			setup: func(t *testing.T, factory *TestDataFactory) string {
				userUID := uuid.New().String()
				// Создаем пользователя с активной подпиской
				_, err := factory.storage.DB.Exec(`INSERT INTO users (uid, username, email, password_hash, role, subscription_status) 
					VALUES ($1, $2, $3, $4, $5, $6)`,
					userUID, "testuser", "test@example.com", "hashedpassword", "user", "active")
				require.NoError(t, err)
				return userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			userUID := tt.setup(t, factory)
			tt.args.userUID = userUID

			err := storage.UpdateStatusCancelForSubscription(tt.args.ctx, tt.args.userUID, tt.args.status)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Проверяем, что статус обновился
				verification := NewTestVerification(storage)
				verification.VerifyUserSubscriptionStatus(t, userUID, "expired")
			}
		})
	}
}

func TestStorage_FindSubscriptionExpiringToday(t *testing.T) {
	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory) error
	}{
		{
			name:      "find users with trial ending today",
			wantCount: 1,
			wantErr:   false,
			setup: func(_ *testing.T, factory *TestDataFactory) error {
				userUID := uuid.New().String()
				_, err := factory.storage.DB.Exec(`INSERT INTO users 
					(uid, username, email, password_hash, role, trial_end_date, subscription_status) 
					VALUES ($1, $2, $3, $4, $5, $6, $7)`,
					userUID, "testuser", "test@example.com", "hashedpassword", "user",
					time.Now().Format("2006-01-02"), "trial")
				return err
			},
		},
		{
			name:      "no users with trial ending today",
			wantCount: 0,
			wantErr:   false,
			setup: func(_ *testing.T, factory *TestDataFactory) error {
				userUID := uuid.New().String()
				_, err := factory.storage.DB.Exec(`INSERT INTO users 
					(uid, username, email, password_hash, role, trial_end_date, subscription_status) 
					VALUES ($1, $2, $3, $4, $5, $6, $7)`,
					userUID, "testuser", "test@example.com", "hashedpassword", "user",
					time.Now().AddDate(0, 0, 1).Format("2006-01-02"), "trial")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			err := tt.setup(t, factory)
			require.NoError(t, err)

			got, err := storage.FindSubscriptionExpiringToday(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCount)
			}
		})
	}
}

func TestStorage_FindOldNextPaymentDate(t *testing.T) {
	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
		setup     func(t *testing.T, factory *TestDataFactory) error
	}{
		{
			name:      "find subscriptions with old next payment date",
			wantCount: 1,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) error {
				userUID := uuid.New().String()
				// Создаем пользователя
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")

				// Создаем подписку с next_payment_date = сегодня
				_, err := factory.storage.DB.Exec(`INSERT INTO subscriptions 
					(service_name, price, username, start_date, counter_months, user_uid, next_payment_date, is_active)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
					"Netflix", 1000.0, "testuser", time.Now().AddDate(0, -1, 0), 12, userUID, time.Now().AddDate(0, 0, -1), true)
				return err
			},
		},
		{
			name:      "no subscriptions with old next payment date",
			wantCount: 0,
			wantErr:   false,
			setup: func(t *testing.T, factory *TestDataFactory) error {
				userUID := uuid.New().String()
				// Создаем пользователя
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")

				// Создаем подписку с next_payment_date = завтра
				_, err := factory.storage.DB.Exec(`INSERT INTO subscriptions 
					(service_name, price, username, start_date, counter_months, user_uid, next_payment_date, is_active)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
					"Netflix", 1000.0, "testuser", time.Now().AddDate(0, -1, 0), 12, userUID, time.Now().AddDate(0, 0, 1), true)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			err := tt.setup(t, factory)
			require.NoError(t, err)

			got, err := storage.FindOldNextPaymentDate(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCount)
			}
		})
	}
}

func TestStorage_UpdateNextPaymentDate(t *testing.T) {
	type args struct {
		ctx   context.Context
		entry *models.Entry
	}

	startDate := time.Now().AddDate(0, -1, 0)
	newPaymentDate := time.Now().AddDate(0, 1, 0)

	tests := []struct {
		name             string
		args             args
		wantRowsAffected int
		wantErr          bool
		setup            func(t *testing.T, factory *TestDataFactory) (int, string)
	}{
		{
			name: "successful update next payment date",
			args: args{
				ctx: context.Background(),
				entry: &models.Entry{
					ID:              0, // будет установлен в setup
					ServiceName:     "Netflix",
					Price:           1000,
					Username:        "testuser",
					StartDate:       startDate,
					CounterMonths:   12,
					UserUID:         "", // будет установлен в setup
					NextPaymentDate: newPaymentDate,
					IsActive:        true,
				},
			},
			wantRowsAffected: 1,
			wantErr:          false,
			setup: func(t *testing.T, factory *TestDataFactory) (int, string) {
				userUID := uuid.New().String()
				// Создаем пользователя
				factory.CreateUser(t, userUID, "testuser", "test@example.com", "hashedpassword", "user")

				// Создаем подписку
				subscriptionID := factory.CreateSubscription(t, "Netflix", 1000.0, "testuser", startDate, 12, userUID, time.Now(), true)
				return subscriptionID, userUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDatabase(t)
			defer cleanup()

			factory := NewTestDataFactory(storage)
			subscriptionID, userUID := tt.setup(t, factory)
			tt.args.entry.ID = subscriptionID
			tt.args.entry.UserUID = userUID

			gotRowsAffected, err := storage.UpdateNextPaymentDate(tt.args.ctx, tt.args.entry)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRowsAffected, gotRowsAffected)
			}

			// Проверяем, что дата обновилась
			var nextPaymentDate time.Time
			err = storage.DB.QueryRow("SELECT next_payment_date FROM subscriptions WHERE id = $1", subscriptionID).
				Scan(&nextPaymentDate)
			require.NoError(t, err)
			// Сравниваем даты с точностью до дня
			assert.Equal(t, newPaymentDate.Year(), nextPaymentDate.Year())
			assert.Equal(t, newPaymentDate.Month(), nextPaymentDate.Month())
			assert.Equal(t, newPaymentDate.Day(), nextPaymentDate.Day())
		})
	}
}
