package paymentlist

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error) {
	args := m.Called(ctx, userUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PaymentToken), args.Error(1)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestPaymentListHandler_ServeHTTP(t *testing.T) {
	paymentTokens := []*models.PaymentToken{
		{ID: 1, UserUID: "user123", Token: "token1"},
		{ID: 2, UserUID: "user123", Token: "token2"},
	}

	tests := []struct {
		name           string
		userUID        string
		setupMocks     func(*MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "success - return payment tokens",
			userUID: "user123",
			setupMocks: func(ps *MockService) {
				ps.On("ListPaymentTokens", mock.Anything, "user123").Return(paymentTokens, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","data":{"list_count":2,"payment tokens":[{"id":1,"user_uid":"user123","token":"token1","created_at":"0001-01-01T00:00:00Z"},{"id":2,"user_uid":"user123","token":"token2","created_at":"0001-01-01T00:00:00Z"}]}}`,
		},
		{
			name:    "success - empty list",
			userUID: "user456",
			setupMocks: func(ps *MockService) {
				ps.On("ListPaymentTokens", mock.Anything, "user456").Return([]*models.PaymentToken{}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","data":{"list_count":0,"payment tokens":[]}}`,
		},
		{
			name:           "missing user UID",
			userUID:        "",
			setupMocks:     func(*MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name:    "service error",
			userUID: "user789",
			setupMocks: func(ps *MockService) {
				ps.On("ListPaymentTokens", mock.Anything, "user789").Return(nil, assert.AnError).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"internal error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentService := new(MockService)
			handler := New(newNoopLogger(), paymentService)

			tt.setupMocks(paymentService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/list", nil)

			ctx := context.WithValue(req.Context(), middlewarectx.UserUID, tt.userUID)
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-id")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())

			paymentService.AssertExpectations(t)
		})
	}
}

func TestPaymentListHandler_New(t *testing.T) {
	logger := newNoopLogger()
	paymentService := new(MockService)

	handler := New(logger, paymentService)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.log)
	assert.Equal(t, paymentService, handler.paymentService)
	assert.NotNil(t, handler.validate)
}

func TestPaymentListHandler_ContextHandling(t *testing.T) {
	tests := []struct {
		name           string
		contextValue   interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid string user UID",
			contextValue:   "user123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "nil context value",
			contextValue:   nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name:           "non-string context value",
			contextValue:   123,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name:           "empty string user UID",
			contextValue:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentService := new(MockService)
			handler := New(newNoopLogger(), paymentService)

			// Настраиваем мок только для успешного случая
			if tt.expectedStatus == http.StatusOK {
				paymentService.On("ListPaymentTokens", mock.Anything, "user123").Return([]*models.PaymentToken{}, nil).Once()
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/list", nil)

			ctx := context.WithValue(req.Context(), middlewarectx.UserUID, tt.contextValue)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			paymentService.AssertExpectations(t)
		})
	}
}

func TestPaymentListHandler_ResponseFormat(t *testing.T) {
	paymentTokens := []*models.PaymentToken{
		{ID: 1, UserUID: "user123", Token: "token1"},
		{ID: 2, UserUID: "user123", Token: "token2"},
		{ID: 3, UserUID: "user123", Token: "token3"},
	}

	paymentService := new(MockService)
	handler := New(newNoopLogger(), paymentService)

	paymentService.On("ListPaymentTokens", mock.Anything, "user123").Return(paymentTokens, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/list", nil)

	ctx := context.WithValue(req.Context(), middlewarectx.UserUID, "user123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем структуру ответа
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "OK", response["status"])
	assert.Contains(t, response, "data")

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["list_count"])
	assert.Contains(t, data, "payment tokens")

	paymentService.AssertExpectations(t)
}
