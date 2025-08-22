package sum

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// MockService реализует интерфейс countsum.Service
type MockService struct {
	mock.Mock
}

func (m *MockService) CountSumWithFilter(ctx context.Context, username string, filter models.DummyFilterSum) (float64, error) {
	args := m.Called(ctx, username, filter)
	return args.Get(0).(float64), args.Error(1)
}

func TestCountSumHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		requestBody    interface{}
		username       string
		setupMock      func(*MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "ошибка валидации - отсутствуют обязательные поля",
			requestBody: models.DummyFilterSum{
				StartDate:     "",
				CounterMonths: 0,
			},
			username:       "testuser",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"field StartDate is a required field, field CounterMonths is a required field"}`,
		},
		{
			name:           "некорректный JSON",
			requestBody:    "not a json",
			username:       "testuser",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid request body"}`,
		},
		{
			name:           "нет авторизации",
			requestBody:    models.DummyFilterSum{StartDate: "2024-01-01", CounterMonths: 6},
			username:       "",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name: "ошибка сервиса",
			requestBody: models.DummyFilterSum{
				StartDate:     "2024-01-01",
				CounterMonths: 6,
			},
			username: "testuser",
			setupMock: func(m *MockService) {
				m.On("CountSumWithFilter", mock.Anything, "testuser", mock.Anything).
					Return(0.0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"could not calculate sum"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockService)
			tt.setupMock(mockSvc)

			handler := New(logger, mockSvc)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/sum", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middlewarectx.User, tt.username)
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-id")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockSvc.AssertExpectations(t)
		})
	}
}
