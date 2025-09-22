package payment

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindPaymentToken(ctx context.Context, userUID string, token string) (int, bool, error) {
	args := m.Called(ctx, userUID, token)
	return args.Int(0), args.Bool(1), args.Error(2)
}

func (m *MockRepository) CreatePaymentToken(ctx context.Context, userUID string, token string) (int, error) {
	args := m.Called(ctx, userUID, token)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error) {
	args := m.Called(ctx, userUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PaymentToken), args.Error(1)
}

func (m *MockRepository) GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID, serviceName string) (string, error) {
	args := m.Called(ctx, userUID, serviceName)
	return args.String(0), args.Error(1)
}

func (m *MockRepository) SavePayment(ctx context.Context, payload *paymentwebhook.Payload, amount int64, userUID string) (int, error) {
	args := m.Called(ctx, payload, amount, userUID)
	return args.Int(0), args.Error(1) 
}

func (m *MockRepository) UpdateStatusActiveForSubscription(ctx context.Context, userUID, status string) error {
	args := m.Called(ctx, userUID, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateStatusCancelForSubscription(ctx context.Context, userUID, status string) error {
	args := m.Called(ctx, userUID, status)
	return args.Error(0)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestService_GetOrCreatePaymentToken(t *testing.T) {
	tests := []struct {
		name          string
		userUID       string
		token         string
		setupMocks    func(*MockRepository)
		expectedID    int
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "token exists - return existing ID",
			userUID: "user123",
			token:   "token123",
			setupMocks: func(r *MockRepository) {
				r.On("FindPaymentToken", mock.Anything, "user123", "token123").Return(42, true, nil).Once()
			},
			expectedID:    42,
			expectedError: false,
		},
		{
			name:    "token not found - create new",
			userUID: "user123",
			token:   "token456",
			setupMocks: func(r *MockRepository) {
				r.On("FindPaymentToken", mock.Anything, "user123", "token456").Return(0, false, nil).Once()
				r.On("CreatePaymentToken", mock.Anything, "user123", "token456").Return(43, nil).Once()
			},
			expectedID:    43,
			expectedError: false,
		},
		{
			name:    "find token error",
			userUID: "user123",
			token:   "token789",
			setupMocks: func(r *MockRepository) {
				r.On("FindPaymentToken", mock.Anything, "user123", "token789").Return(0, false, errors.New("db error")).Once()
			},
			expectedID:    0,
			expectedError: true,
			errorMessage:  "failed to find token: db error",
		},
		{
			name:    "create token error",
			userUID: "user123",
			token:   "token999",
			setupMocks: func(r *MockRepository) {
				r.On("FindPaymentToken", mock.Anything, "user123", "token999").Return(0, false, nil).Once()
				r.On("CreatePaymentToken", mock.Anything, "user123", "token999").Return(0, errors.New("create error")).Once()
			},
			expectedID:    0,
			expectedError: true,
			errorMessage:  "failed to create token: create error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			service := New(repo, newNoopLogger())

			tt.setupMocks(repo)

			result, err := service.GetOrCreatePaymentToken(context.Background(), tt.userUID, tt.token)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
				assert.Equal(t, tt.expectedID, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestService_ListPaymentTokens(t *testing.T) {
	expectedTokens := []*models.PaymentToken{
		{ID: 1, UserUID: "user123", Token: "token1"},
		{ID: 2, UserUID: "user123", Token: "token2"},
	}

	tests := []struct {
		name           string
		userUID        string
		setupMocks     func(*MockRepository)
		expectedTokens []*models.PaymentToken
		expectedError  bool
		errorMessage   string
	}{
		{
			name:    "success - return tokens",
			userUID: "user123",
			setupMocks: func(r *MockRepository) {
				r.On("ListPaymentTokens", mock.Anything, "user123").Return(expectedTokens, nil).Once()
			},
			expectedTokens: expectedTokens,
			expectedError:  false,
		},
		{
			name:    "empty result",
			userUID: "user456",
			setupMocks: func(r *MockRepository) {
				r.On("ListPaymentTokens", mock.Anything, "user456").Return([]*models.PaymentToken{}, nil).Once()
			},
			expectedTokens: []*models.PaymentToken{},
			expectedError:  false,
		},
		{
			name:    "repository error",
			userUID: "user789",
			setupMocks: func(r *MockRepository) {
				r.On("ListPaymentTokens", mock.Anything, "user789").Return(nil, errors.New("db error")).Once()
			},
			expectedTokens: nil,
			expectedError:  true,
			errorMessage:   "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			service := New(repo, newNoopLogger())

			tt.setupMocks(repo)

			result, err := service.ListPaymentTokens(context.Background(), tt.userUID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTokens, result)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestService_GetActiveSubscriptionIDByUserUID(t *testing.T) {
	tests := []struct {
		name          string
		userUID       string
		setupMocks    func(*MockRepository)
		expectedID    string
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "success - return subscription ID",
			userUID: "user123",
			setupMocks: func(r *MockRepository) {
				r.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user123", "Subscription-Aggregator").Return("sub123", nil).Once()
			},
			expectedID:    "sub123",
			expectedError: false,
		},
		{
			name:    "no active subscription",
			userUID: "user456",
			setupMocks: func(r *MockRepository) {
				r.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user456", "Subscription-Aggregator").Return("", nil).Once()
			},
			expectedID:    "",
			expectedError: false,
		},
		{
			name:    "repository error",
			userUID: "user789",
			setupMocks: func(r *MockRepository) {
				r.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user789", "Subscription-Aggregator").Return("", errors.New("db error")).Once()
			},
			expectedID:    "",
			expectedError: true,
			errorMessage:  "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			service := New(repo, newNoopLogger())

			tt.setupMocks(repo)

			result, err := service.GetActiveSubscriptionIDByUserUID(context.Background(), tt.userUID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
				assert.Equal(t, tt.expectedID, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestService_SavePayment(t *testing.T) {
	payload := &paymentwebhook.Payload{
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
			ID:     "payment123",
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
				ID: "card123",
			},
			Metadata: map[string]string{
				"user_uid": "user123",
			},
		},
	}

	tests := []struct {
		name          string
		payload       *paymentwebhook.Payload
		setupMocks    func(*MockRepository)
		expectedID    int
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "success - save payment",
			payload: payload,
			setupMocks: func(r *MockRepository) {
				r.On("SavePayment", mock.Anything, payload).Return(42, nil).Once()
			},
			expectedID:    42,
			expectedError: false,
		},
		{
			name:    "repository error",
			payload: payload,
			setupMocks: func(r *MockRepository) {
				r.On("SavePayment", mock.Anything, payload).Return(0, errors.New("db error")).Once()
			},
			expectedID:    0,
			expectedError: true,
			errorMessage:  "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			service := New(repo, newNoopLogger())

			tt.setupMocks(repo)

			result, err := service.SavePayment(context.Background(), tt.payload)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
				assert.Equal(t, tt.expectedID, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestService_UpdateStatusActiveForSubscription(t *testing.T) {
	tests := []struct {
		name          string
		userUID       string
		setupMocks    func(*MockRepository)
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "success - update status to active",
			userUID: "user123",
			setupMocks: func(r *MockRepository) {
				r.On("UpdateStatusActiveForSubscription", mock.Anything, "user123", "active").Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name:    "repository error",
			userUID: "user456",
			setupMocks: func(r *MockRepository) {
				r.On("UpdateStatusActiveForSubscription", mock.Anything, "user456", "active").Return(errors.New("db error")).Once()
			},
			expectedError: true,
			errorMessage:  "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			service := New(repo, newNoopLogger())

			tt.setupMocks(repo)

			err := service.UpdateStatusActiveForSubscription(context.Background(), tt.userUID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestService_UpdateStatusExpireForSubscription(t *testing.T) {
	tests := []struct {
		name          string
		userUID       string
		setupMocks    func(*MockRepository)
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "success - update status to expire",
			userUID: "user123",
			setupMocks: func(r *MockRepository) {
				r.On("UpdateStatusCancelForSubscription", mock.Anything, "user123", "expire").Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name:    "repository error",
			userUID: "user456",
			setupMocks: func(r *MockRepository) {
				r.On("UpdateStatusCancelForSubscription", mock.Anything, "user456", "expire").Return(errors.New("db error")).Once()
			},
			expectedError: true,
			errorMessage:  "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			service := New(repo, newNoopLogger())

			tt.setupMocks(repo)

			err := service.UpdateStatusExpireForSubscription(context.Background(), tt.userUID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}
