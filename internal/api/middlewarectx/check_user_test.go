package middlewarectx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) GetSubscriptionStatus(ctx context.Context, userUID string) (string, error) {
	args := m.Called(ctx, userUID)
	return args.String(0), args.Error(1)
}

// MockSubscriptionService реализует SubscriptionServiceInterface для тестирования

func newNoopLoggerCheck() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestSubscriptionStatusMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		userUID        string
		setupMocks     func(*MockSubscriptionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "success - active subscription",
			userUID: "user123",
			setupMocks: func(ss *MockSubscriptionService) {
				ss.On("GetSubscriptionStatus", mock.Anything, "user123").Return("active", nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "success - trial subscription",
			userUID: "user456",
			setupMocks: func(ss *MockSubscriptionService) {
				ss.On("GetSubscriptionStatus", mock.Anything, "user456").Return("trial", nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "forbidden - expired subscription",
			userUID: "user789",
			setupMocks: func(ss *MockSubscriptionService) {
				ss.On("GetSubscriptionStatus", mock.Anything, "user789").Return("expired", nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"status":"Error","error":"subscription expired, access denied"}` + "\n",
		},
		{
			name:           "unauthorized - missing user UID",
			userUID:        "",
			setupMocks:     func(*MockSubscriptionService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"user identification missing"}` + "\n",
		},
		{
			name:    "internal server error - service error",
			userUID: "user999",
			setupMocks: func(ss *MockSubscriptionService) {
				ss.On("GetSubscriptionStatus", mock.Anything, "user999").Return("", errors.New("service error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"internal service error"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subService := new(MockSubscriptionService)
			logger := newNoopLoggerCheck()
			middleware := SubscriptionStatusMiddleware(logger, subService)

			tt.setupMocks(subService)

			// Создаем тестовый handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("success")); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Устанавливаем user UID в контекст
			ctx := context.WithValue(req.Context(), UserUID, tt.userUID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			middleware(testHandler).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			} else {
				assert.Equal(t, "success", w.Body.String())
			}

			subService.AssertExpectations(t)
		})
	}
}

func TestSubscriptionStatusMiddleware_ContextHandling(t *testing.T) {
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
			expectedBody:   `{"status":"Error","error":"user identification missing"}` + "\n",
		},
		{
			name:           "non-string context value",
			contextValue:   123,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"user identification missing"}` + "\n",
		},
		{
			name:           "empty string user UID",
			contextValue:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"user identification missing"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subService := new(MockSubscriptionService)
			logger := newNoopLoggerCheck()
			middleware := SubscriptionStatusMiddleware(logger, subService)

			// Настраиваем мок только для успешного случая
			if tt.expectedStatus == http.StatusOK {
				subService.On("GetSubscriptionStatus", mock.Anything, "user123").Return("active", nil).Once()
			}

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("success")); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			ctx := context.WithValue(req.Context(), UserUID, tt.contextValue)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			middleware(testHandler).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			} else {
				assert.Equal(t, "success", w.Body.String())
			}

			subService.AssertExpectations(t)
		})
	}
}

func TestSubscriptionStatusMiddleware_SubscriptionStatuses(t *testing.T) {
	subService := new(MockSubscriptionService)
	logger := newNoopLoggerCheck()
	middleware := SubscriptionStatusMiddleware(logger, subService)

	statuses := []struct {
		status         string
		expectedStatus int
		expectedBody   string
	}{
		{"active", http.StatusOK, "success"},
		{"trial", http.StatusOK, "success"},
		{"pending", http.StatusOK, "success"},
		{"expired", http.StatusForbidden, `{"status":"Error","error":"subscription expired, access denied"}` + "\n"},
		{"cancelled", http.StatusOK, "success"}, // не expired, поэтому OK
	}

	for _, status := range statuses {
		t.Run("status_"+status.status, func(t *testing.T) {
			subService.On("GetSubscriptionStatus", mock.Anything, "user123").Return(status.status, nil).Once()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("success")); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			ctx := context.WithValue(req.Context(), UserUID, "user123")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			middleware(testHandler).ServeHTTP(w, req)

			assert.Equal(t, status.expectedStatus, w.Code)
			assert.Equal(t, status.expectedBody, w.Body.String())

			subService.AssertExpectations(t)
		})
	}
}

func TestSubscriptionStatusMiddleware_HandlerNotCalledOnError(t *testing.T) {
	subService := new(MockSubscriptionService)
	logger := newNoopLoggerCheck()
	middleware := SubscriptionStatusMiddleware(logger, subService)

	subService.On("GetSubscriptionStatus", mock.Anything, "user123").Return("", errors.New("service error")).Once()

	var handlerCalled bool
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx := context.WithValue(req.Context(), UserUID, "user123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	middleware(testHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.False(t, handlerCalled, "Handler should not be called when there's an error")

	subService.AssertExpectations(t)
}
