package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// MockService реализует интерфейс update.Service
type MockService struct {
	mock.Mock
}

func (m *MockService) UpdateEntry(ctx context.Context, req models.DummyEntry, id int, username string) (int, error) {
	args := m.Called(ctx, req, id, username)
	return args.Int(0), args.Error(1)
}

func TestUpdateHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		url            string
		requestBody    interface{}
		username       string
		setupMock      func(*MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "успешное обновление подписки",
			url:  "/subscriptions/123",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         15,
				StartDate:     "01-2024",
				CounterMonths: 6,
			},
			username: "testuser",
			setupMock: func(m *MockService) {
				m.On("Update", mock.Anything, mock.AnythingOfType("models.DummyEntry"), 123, "testuser").
					Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"updated_count":1`,
		},
		{
			name:           "некорректный JSON",
			url:            "/subscriptions/123",
			requestBody:    "not a json",
			username:       "testuser",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"failed to decode request"}`,
		},
		{
			name: "ошибка валидации",
			url:  "/subscriptions/123",
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
			name: "отсутствует авторизация",
			url:  "/subscriptions/123",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         15,
				StartDate:     "01-2024",
				CounterMonths: 6,
			},
			username:       "",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name: "некорректный id в url",
			url:  "/subscriptions/abc",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         15,
				StartDate:     "01-2024",
				CounterMonths: 6,
			},
			username:       "testuser",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"failed to decode id from url"}`,
		},
		{
			name: "ошибка сервиса",
			url:  "/subscriptions/123",
			requestBody: models.DummyEntry{
				ServiceName:   "Netflix",
				Price:         15,
				StartDate:     "01-2024",
				CounterMonths: 6,
			},
			username: "testuser",
			setupMock: func(m *MockService) {
				m.On("Update", mock.Anything, mock.AnythingOfType("models.DummyEntry"), 123, "testuser").
					Return(0, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"could not update subscription"}`,
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
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPut, tt.url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), middlewarectx.User, tt.username)
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-id")
			req = req.WithContext(ctx)

			// Устанавливаем URL параметр id для chi
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strings.TrimPrefix(tt.url, "/subscriptions/"))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			mockService.AssertExpectations(t)
		})
	}
}
