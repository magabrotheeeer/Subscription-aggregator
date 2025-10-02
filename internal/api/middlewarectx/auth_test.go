package middlewarectx

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Типизированные ключи для контекста
type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

type MockAuthClient struct {
	mock.Mock
}

func (m *MockAuthClient) ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authpb.ValidateTokenResponse), args.Error(1)
}

func (m *MockAuthClient) GetUser(ctx context.Context, userUID string) (*models.User, error) {
	args := m.Called(ctx, userUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func newNoopLoggerAuth() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestJWTMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*MockAuthClient)
		expectedStatus int
		expectedBody   string
		expectedCtx    map[Key]interface{}
	}{
		{
			name:       "success - valid token",
			authHeader: "Bearer valid_token_123",
			setupMocks: func(ac *MockAuthClient) {
				ac.On("ValidateToken", mock.Anything, "valid_token_123").Return(&authpb.ValidateTokenResponse{
					Valid:    true,
					Username: "testuser",
					Role:     "user",
					Useruid:  "user123",
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedCtx: map[Key]interface{}{
				User:    "testuser",
				Role:    "user",
				UserUID: "user123",
			},
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			setupMocks:     func(*MockAuthClient) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"missing or invalid authorization header"}`,
		},
		{
			name:           "invalid authorization header format",
			authHeader:     "InvalidFormat token123",
			setupMocks:     func(*MockAuthClient) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"missing or invalid authorization header"}`,
		},
		{
			name:       "empty bearer token",
			authHeader: "Bearer ",
			setupMocks: func(ac *MockAuthClient) {
				ac.On("ValidateToken", mock.Anything, "").Return(&authpb.ValidateTokenResponse{
					Valid: false,
				}, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"invalid or expired token"}`,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid_token",
			setupMocks: func(ac *MockAuthClient) {
				ac.On("ValidateToken", mock.Anything, "invalid_token").Return(&authpb.ValidateTokenResponse{
					Valid: false,
				}, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"invalid or expired token"}`,
		},
		{
			name:       "auth service error",
			authHeader: "Bearer error_token",
			setupMocks: func(ac *MockAuthClient) {
				ac.On("ValidateToken", mock.Anything, "error_token").Return(nil, assert.AnError).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"status":"Error","error":"invalid or expired token"}`,
		},
		{
			name:       "valid token with admin role",
			authHeader: "Bearer admin_token",
			setupMocks: func(ac *MockAuthClient) {
				ac.On("ValidateToken", mock.Anything, "admin_token").Return(&authpb.ValidateTokenResponse{
					Valid:    true,
					Username: "admin",
					Role:     "admin",
					Useruid:  "admin123",
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedCtx: map[Key]interface{}{
				User:    "admin",
				Role:    "admin",
				UserUID: "admin123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authClient := new(MockAuthClient)
			logger := newNoopLoggerAuth()
			middleware := JWTMiddleware(logger, authClient)

			tt.setupMocks(authClient)

			// Создаем тестовый handler, который проверяет контекст
			var capturedCtx context.Context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedCtx = r.Context()
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("success")); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Добавляем request ID в контекст
			ctx := context.WithValue(req.Context(), requestIDKey, "test-req-id")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			middleware(testHandler).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			// Проверяем контекст только для успешных случаев
			if tt.expectedStatus == http.StatusOK && tt.expectedCtx != nil {
				assert.NotNil(t, capturedCtx)
				for key, expectedValue := range tt.expectedCtx {
					actualValue := capturedCtx.Value(key)
					assert.Equal(t, expectedValue, actualValue)
				}
			}

			authClient.AssertExpectations(t)
		})
	}
}

func TestJWTMiddleware_ContextValues(t *testing.T) {
	authClient := new(MockAuthClient)
	logger := newNoopLoggerAuth()
	middleware := JWTMiddleware(logger, authClient)

	authClient.On("ValidateToken", mock.Anything, "test_token").Return(&authpb.ValidateTokenResponse{
		Valid:    true,
		Username: "testuser",
		Role:     "user",
		Useruid:  "user123",
	}, nil).Once()

	var capturedCtx context.Context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test_token")

	ctx := context.WithValue(req.Context(), requestIDKey, "test-req-id")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	middleware(testHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedCtx)

	// Проверяем, что все значения правильно установлены в контексте
	assert.Equal(t, "testuser", capturedCtx.Value(User))
	assert.Equal(t, "user", capturedCtx.Value(Role))
	assert.Equal(t, "user123", capturedCtx.Value(UserUID))

	authClient.AssertExpectations(t)
}

func TestJWTMiddleware_EmptyToken(t *testing.T) {
	authClient := new(MockAuthClient)
	logger := newNoopLoggerAuth()
	middleware := JWTMiddleware(logger, authClient)

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("Handler should not be called for invalid token")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	middleware(testHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.JSONEq(t, `{"status":"Error","error":"missing or invalid authorization header"}`, w.Body.String())

	authClient.AssertExpectations(t)
}
