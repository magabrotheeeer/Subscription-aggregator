package list

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// MockService реализует интерфейс list.Service
type MockService struct {
	mock.Mock
}

func (m *MockService) ListEntrys(ctx context.Context, username, role string, limit, offset int) ([]*models.Entry, error) {
	args := m.Called(ctx, username, role, limit, offset)
	return args.Get(0).([]*models.Entry), args.Error(1)
}

func TestListHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		queryParams    string
		username       string
		role           string
		setupMock      func(*MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "успешный список с дефолтной пагинацией",
			queryParams: "",
			username:    "testuser",
			role:        "user",
			setupMock: func(m *MockService) {
				entries := []*models.Entry{
					{ServiceName: "Netflix", Price: 10, Username: "testuser", CounterMonths: 3},
					{ServiceName: "Spotify", Price: 5, Username: "testuser", CounterMonths: 1},
				}
				m.On("ListEntrys", mock.Anything, "testuser", "user", 10, 0).
					Return(entries, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"list_count":2`,
		},
		{
			name:        "кастомная пагинация",
			queryParams: "?limit=5&offset=3",
			username:    "testuser",
			role:        "user",
			setupMock: func(m *MockService) {
				m.On("ListEntrys", mock.Anything, "testuser", "user", 5, 3).
					Return([]*models.Entry{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"list_count":0`,
		},
		{
			name:        "некорректный параметр limit",
			queryParams: "?limit=abc",
			username:    "testuser",
			role:        "user",
			setupMock: func(m *MockService) {
				m.On("ListEntrys", mock.Anything, "testuser", "user", 10, 0).
					Return([]*models.Entry{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"list_count":0`,
		},
		{
			name:           "нет авторизации (username)",
			queryParams:    "",
			username:       "",
			role:           "user",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name:           "нет роли в контексте",
			queryParams:    "",
			username:       "testuser",
			role:           "",
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"unauthorized"}`,
		},
		{
			name:        "ошибка сервиса",
			queryParams: "",
			username:    "testuser",
			role:        "admin",
			setupMock: func(m *MockService) {
				m.On("ListEntrys", mock.Anything, "testuser", "admin", 10, 0).
					Return([]*models.Entry{}, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to list"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := New(logger, mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/list"+tt.queryParams, nil)

			ctx := context.WithValue(req.Context(), middlewarectx.User, tt.username)
			ctx = context.WithValue(ctx, middlewarectx.Role, tt.role)
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-id")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.True(t, strings.Contains(w.Body.String(), tt.expectedBody),
				"response body should contain %s, got %s", tt.expectedBody, w.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}
