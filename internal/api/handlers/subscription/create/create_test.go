package create

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

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// MockService реализует интерфейс create.Service
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateEntry(ctx context.Context, userName string, userUID string, req models.DummyEntry) (int, error) {
	args := m.Called(ctx, userName, userUID, req)
	return args.Int(0), args.Error(1)
}

func TestCreateHandler(t *testing.T) {
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
			name: "успешное создание подписки",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         10,
				StartDate:     "01-2024",
				CounterMonths: 12,
			},
			username: "testuser",
			setupMock: func(m *MockService) {
				m.On("CreateEntry", mock.Anything, "testuser", "user123", mock.AnythingOfType("models.DummyEntry")).
					Return(123, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","data":{"last_added_id":123}}`,
		},
		{
			name: "невалидные данные",
			requestBody: models.DummyEntry{
				ServiceName:   "",
				Price:         0,
				StartDate:     "",
				CounterMonths: 0,
			},
			username:       "testuser",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   `{"status":"Error","error":"field ServiceName is a required field, field Price is a required field, field StartDate is a required field, field CounterMonths is a required field"}`,
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
			name: "отсутствует авторизация",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         10,
				StartDate:     "01-2024",
				CounterMonths: 12,
			},
			username:       "",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name: "ошибка сервиса",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         10,
				StartDate:     "01-2024",
				CounterMonths: 12,
			},
			username: "testuser",
			setupMock: func(m *MockService) {
				m.On("CreateEntry", mock.Anything, "testuser", "user123", mock.AnythingOfType("models.DummyEntry")).
					Return(0, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"could not create subscription"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := New(logger, mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middlewarectx.User, tt.username)
			ctx = context.WithValue(ctx, middlewarectx.UserUID, "user123")
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-id")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}
