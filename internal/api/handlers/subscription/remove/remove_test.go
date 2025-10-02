package remove

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
)

// MockService реализует интерфейс remove.Service
type MockService struct {
	mock.Mock
}

func (m *MockService) RemoveEntry(ctx context.Context, id int) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

func TestRemoveHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		url            string
		mockID         int
		setupMock      func(*MockService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "успешное удаление",
			url:    "/subscriptions/123",
			mockID: 123,
			setupMock: func(m *MockService) {
				m.On("RemoveEntry", mock.Anything, 123).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"deleted_count":1`,
		},
		{
			name:           "некорректный id",
			url:            "/subscriptions/abc",
			mockID:         0,
			setupMock:      func(_ *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid id"}`,
		},
		{
			name:   "ошибка сервиса",
			url:    "/subscriptions/777",
			mockID: 777,
			setupMock: func(m *MockService) {
				m.On("RemoveEntry", mock.Anything, 777).Return(0, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to delete subscription"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := New(logger, mockService)

			req := httptest.NewRequest(http.MethodDelete, tt.url, nil)
			// Устанавливаем URL param для ID
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
