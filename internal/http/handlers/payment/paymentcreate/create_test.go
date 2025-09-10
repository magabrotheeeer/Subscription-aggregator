package paymentcreate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/paymentprovider"
)

type MockProviderClient struct {
	mock.Mock
}

func (m *MockProviderClient) CreatePayment(reqParams paymentprovider.CreatePaymentRequest) (*paymentprovider.CreatePaymentResponse, error) {
	args := m.Called(reqParams)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*paymentprovider.CreatePaymentResponse), args.Error(1)
}

type MockService struct {
	mock.Mock
}

func (m *MockService) GetOrCreatePaymentToken(ctx context.Context, userUID string, token string) (int, error) {
	args := m.Called(ctx, userUID, token)
	return args.Int(0), args.Error(1)
}

func (m *MockService) GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID string) (string, error) {
	args := m.Called(ctx, userUID)
	return args.String(0), args.Error(1)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestPaymentCreateHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		userUID        string
		setupMocks     func(*MockProviderClient, *MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success - create payment",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "token123",
			},
			userUID: "user123",
			setupMocks: func(pc *MockProviderClient, ps *MockService) {
				ps.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user123").Return("sub123", nil).Once()
				ps.On("GetOrCreatePaymentToken", mock.Anything, "user123", "token123").Return(42, nil).Once()
				pc.On("CreatePayment", mock.MatchedBy(func(req paymentprovider.CreatePaymentRequest) bool {
					return req.PaymentToken == "token123" &&
						req.Amount.Value == "200.00" &&
						req.Amount.Currency == "RUB" &&
						req.Metadata["user_uid"] == "user123" &&
						req.Metadata["subscription_id"] == "sub123"
				})).Return(&paymentprovider.CreatePaymentResponse{
					ID:     "payment123",
					Status: "succeeded",
					Amount: paymentprovider.Amount{
						Value:    "200.00",
						Currency: "RUB",
					},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"payment123","status":"succeeded","amount":{"value":"200.00","currency":"RUB"},"created_at":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:           "invalid JSON",
			requestBody:    "not a json",
			userUID:        "user123",
			setupMocks:     func(*MockProviderClient, *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid request body"}`,
		},
		{
			name: "missing payment method token",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "",
			},
			userUID:        "user123",
			setupMocks:     func(*MockProviderClient, *MockService) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   `{"status":"Error","error":"field PaymentMethodToken is a required field"}`,
		},
		{
			name: "missing user UID",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "token123",
			},
			userUID:        "",
			setupMocks:     func(*MockProviderClient, *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name: "get active subscription error",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "token123",
			},
			userUID: "user123",
			setupMocks: func(_ *MockProviderClient, ps *MockService) {
				ps.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user123").Return("", errors.New("subscription not found")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"internal error"}`,
		},
		{
			name: "get or create payment token error",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "token123",
			},
			userUID: "user123",
			setupMocks: func(_ *MockProviderClient, ps *MockService) {
				ps.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user123").Return("sub123", nil).Once()
				ps.On("GetOrCreatePaymentToken", mock.Anything, "user123", "token123").Return(0, errors.New("token error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"internal error"}`,
		},
		{
			name: "provider client error",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "token123",
			},
			userUID: "user123",
			setupMocks: func(pc *MockProviderClient, ps *MockService) {
				ps.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user123").Return("sub123", nil).Once()
				ps.On("GetOrCreatePaymentToken", mock.Anything, "user123", "token123").Return(42, nil).Once()
				pc.On("CreatePayment", mock.Anything).Return(nil, errors.New("provider error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"payment provider error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerClient := new(MockProviderClient)
			paymentService := new(MockService)
			handler := New(newNoopLogger(), providerClient, paymentService)

			tt.setupMocks(providerClient, paymentService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/payment", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middlewarectx.UserUID, tt.userUID)
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-id")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())

			providerClient.AssertExpectations(t)
			paymentService.AssertExpectations(t)
		})
	}
}

func TestPaymentCreateHandler_New(t *testing.T) {
	logger := newNoopLogger()
	providerClient := new(MockProviderClient)
	paymentService := new(MockService)

	handler := New(logger, providerClient, paymentService)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.log)
	assert.Equal(t, providerClient, handler.providerClient)
	assert.Equal(t, paymentService, handler.paymentService)
	assert.NotNil(t, handler.validate)
}

func TestPaymentCreateHandler_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CreatePaymentMethodRequestApp
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "valid request",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "valid_token_123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty token",
			requestBody: CreatePaymentMethodRequestApp{
				PaymentMethodToken: "",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   `{"status":"Error","error":"field PaymentMethodToken is a required field"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerClient := new(MockProviderClient)
			paymentService := new(MockService)
			handler := New(newNoopLogger(), providerClient, paymentService)

			// Настраиваем моки для успешного случая
			if tt.expectedStatus == http.StatusOK {
				paymentService.On("GetActiveSubscriptionIDByUserUID", mock.Anything, "user123").Return("sub123", nil).Once()
				paymentService.On("GetOrCreatePaymentToken", mock.Anything, "user123", tt.requestBody.PaymentMethodToken).Return(42, nil).Once()
				providerClient.On("CreatePayment", mock.Anything).Return(&paymentprovider.CreatePaymentResponse{
					ID:     "payment123",
					Status: "succeeded",
				}, nil).Once()
			}

			body, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/payment", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middlewarectx.UserUID, "user123")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			providerClient.AssertExpectations(t)
			paymentService.AssertExpectations(t)
		})
	}
}
