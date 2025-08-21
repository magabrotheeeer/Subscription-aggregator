package read

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// MockService реализует интерфейс read.Service
type MockService struct {
	mock.Mock
}

func (m *MockService) Read(ctx context.Context, id int) (*models.Entry, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*models.Entry), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestReadHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		url            string
		mockID         int
		username       string // not required here, but could be added if auth is implemented
		setupMock      func(*MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "успешное чтение подписки",
			url:    "/subscriptions/123",
			mockID: 123,
			setupMock: func(m *MockService) {
				entry := &models.Entry{
					ServiceName:   "Netflix",
					Price:         10,
					Username:      "testuser",
					CounterMonths: 6,
				}
				m.On("Read", mock.Anything, 123).Return(entry, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"ServiceName":"Netflix"`,
		},
		{
			name:           "некорректный id в URL",
			url:            "/subscriptions/abc",
			mockID:         0,
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusOK, // Because handler does not call WriteHeader, renders JSON with default 200
			expectedBody:   `{"status":"Error","error":"failed to decode id from url"}`,
		},
		{
			name:   "ошибка сервиса чтения",
			url:    "/subscriptions/777",
			mockID: 777,
			setupMock: func(m *MockService) {
				m.On("Read", mock.Anything, 777).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"could not read subscription"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := New(logger, mockService)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			// Устанавливаем URL params с помощью роутера chi
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strings.TrimPrefix(tt.url, "/subscriptions/"))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.True(t, strings.Contains(w.Body.String(), tt.expectedBody),
				"response body should contain %s, got %s", tt.expectedBody, w.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}
