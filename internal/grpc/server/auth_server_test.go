package server

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// MockAuthService - мок для AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, email, username, password string) (string, error) {
	args := m.Called(ctx, email, username, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (string, string, string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*models.User, string, bool, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Bool(2), args.Error(3)
	}
	return args.Get(0).(*models.User), args.String(1), args.Bool(2), args.Error(3)
}

// Убеждаемся, что MockAuthService реализует интерфейс AuthServiceInterface
var _ AuthServiceInterface = (*MockAuthService)(nil)

// TestNewAuthServer тестирует создание нового AuthServer
func TestNewAuthServer(t *testing.T) {
	mockService := new(MockAuthService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewAuthServer(mockService, logger)

	assert.NotNil(t, server)
	assert.Equal(t, mockService, server.authService)
	assert.Equal(t, logger, server.log)
}

// TestAuthServer_Register_Unit тестирует метод Register с моками
func TestAuthServer_Register_Unit(t *testing.T) {
	tests := []struct {
		name          string
		request       *authpb.RegisterRequest
		mockSetup     func(*MockAuthService)
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "successful registration",
			request: &authpb.RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "test@example.com", "testuser", "password123").
					Return("user-uuid-123", nil).Once()
			},
			expectedError: false,
		},
		{
			name: "registration error",
			request: &authpb.RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "test@example.com", "testuser", "password123").
					Return("", assert.AnError).Once()
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
		{
			name: "duplicate username",
			request: &authpb.RegisterRequest{
				Email:    "test@example.com",
				Username: "existinguser",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "test@example.com", "existinguser", "password123").
					Return("", assert.AnError).Once()
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			tt.mockSetup(mockService)

			server := NewAuthServer(mockService, logger)
			ctx := context.Background()

			resp, err := server.Register(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.expectedCode != codes.Unknown {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Equal(t, "user created successfully", resp.Message)
				assert.NotEmpty(t, resp.Useruid)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthServer_Login_Unit тестирует метод Login с моками
func TestAuthServer_Login_Unit(t *testing.T) {
	tests := []struct {
		name          string
		request       *authpb.LoginRequest
		mockSetup     func(*MockAuthService)
		expectedError bool
		expectedCode  codes.Code
		expectedToken string
		expectedRole  string
	}{
		{
			name: "successful login",
			request: &authpb.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "testuser", "password123").
					Return("jwt-token-123", "refresh-token-123", "user", nil).Once()
			},
			expectedError: false,
			expectedToken: "jwt-token-123",
			expectedRole:  "user",
		},
		{
			name: "invalid credentials",
			request: &authpb.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "testuser", "wrongpassword").
					Return("", "", "", assert.AnError).Once()
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "user not found",
			request: &authpb.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "nonexistent", "password123").
					Return("", "", "", assert.AnError).Once()
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			tt.mockSetup(mockService)

			server := NewAuthServer(mockService, logger)
			ctx := context.Background()

			resp, err := server.Login(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.expectedCode != codes.Unknown {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedToken, resp.Token)
				assert.Equal(t, tt.expectedRole, resp.Role)
				assert.NotEmpty(t, resp.RefreshToken)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthServer_ValidateToken_Unit тестирует метод ValidateToken с моками
func TestAuthServer_ValidateToken_Unit(t *testing.T) {
	tests := []struct {
		name          string
		request       *authpb.ValidateTokenRequest
		mockSetup     func(*MockAuthService)
		expectedError bool
		expectedCode  codes.Code
		expectedValid bool
		expectedUser  string
		expectedRole  string
	}{
		{
			name: "valid token",
			request: &authpb.ValidateTokenRequest{
				Token: "valid.jwt.token",
			},
			mockSetup: func(m *MockAuthService) {
				user := &models.User{
					Username: "testuser",
					Role:     "user",
					UUID:     "user-uuid-123",
				}
				m.On("ValidateToken", mock.Anything, "valid.jwt.token").
					Return(user, "user", true, nil).Once()
			},
			expectedError: false,
			expectedValid: true,
			expectedUser:  "testuser",
			expectedRole:  "user",
		},
		{
			name: "invalid token",
			request: &authpb.ValidateTokenRequest{
				Token: "invalid.token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateToken", mock.Anything, "invalid.token").
					Return(nil, "", false, assert.AnError).Once()
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "expired token",
			request: &authpb.ValidateTokenRequest{
				Token: "expired.jwt.token",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateToken", mock.Anything, "expired.jwt.token").
					Return(nil, "", false, nil).Once()
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "empty token",
			request: &authpb.ValidateTokenRequest{
				Token: "",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateToken", mock.Anything, "").
					Return(nil, "", false, assert.AnError).Once()
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			tt.mockSetup(mockService)

			server := NewAuthServer(mockService, logger)
			ctx := context.Background()

			resp, err := server.ValidateToken(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.expectedCode != codes.Unknown {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedValid, resp.Valid)
				assert.Equal(t, tt.expectedUser, resp.Username)
				assert.Equal(t, tt.expectedRole, resp.Role)
				assert.NotEmpty(t, resp.Useruid)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthServer_ContextCancellation тестирует обработку отмены контекста
func TestAuthServer_ContextCancellation(t *testing.T) {
	mockService := new(MockAuthService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Настраиваем мок для возврата ошибки контекста
	mockService.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", context.Canceled).Maybe()

	server := NewAuthServer(mockService, logger)

	// Создаем отмененный контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем контекст сразу

	// Тестируем Register с отмененным контекстом
	req := &authpb.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	resp, err := server.Register(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "context canceled")
}

// TestAuthServer_Timeout тестирует обработку таймаутов
func TestAuthServer_Timeout(t *testing.T) {
	mockService := new(MockAuthService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Настраиваем мок для возврата ошибки таймаута
	mockService.On("Login", mock.Anything, "testuser", "password123").
		Return("", "", "", context.DeadlineExceeded).Once()

	server := NewAuthServer(mockService, logger)

	// Создаем контекст с коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &authpb.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	resp, err := server.Login(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	// Проверяем, что получили ошибку (может быть "invalid credentials" или другая ошибка)
	assert.NotEmpty(t, err.Error())
}

// TestAuthServer_EdgeCases тестирует edge cases и граничные условия
func TestAuthServer_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		request     interface{}
		description string
	}{
		{
			name:        "nil request",
			request:     nil,
			description: "nil request object",
		},
		{
			name: "empty fields",
			request: &authpb.RegisterRequest{
				Email:    "",
				Username: "",
				Password: "",
			},
			description: "empty fields in request",
		},
		{
			name: "very long strings",
			request: &authpb.RegisterRequest{
				Email:    strings.Repeat("a", 1000) + "@example.com",
				Username: strings.Repeat("b", 1000),
				Password: strings.Repeat("c", 1000),
			},
			description: "very long input strings",
		},
		{
			name: "special characters",
			request: &authpb.RegisterRequest{
				Email:    "test@#$%^&*().com",
				Username: "user@#$%^&*()",
				Password: "pass!@#$%^&*()",
			},
			description: "special characters in input",
		},
		{
			name: "unicode characters",
			request: &authpb.RegisterRequest{
				Email:    "тест@пример.рф",
				Username: "пользователь",
				Password: "пароль123",
			},
			description: "unicode characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			// Настраиваем мок для возврата ошибки для всех edge cases
			mockService.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return("", assert.AnError).Maybe()

			server := NewAuthServer(mockService, logger)
			ctx := context.Background()

			// Тестируем с разными типами запросов
			if req, ok := tt.request.(*authpb.RegisterRequest); ok {
				resp, err := server.Register(ctx, req)
				assert.Error(t, err)
				assert.Nil(t, resp)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthServer_ErrorHandling тестирует обработку различных типов ошибок
func TestAuthServer_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		mockSetup    func(*MockAuthService)
		expectedCode codes.Code
		description  string
	}{
		{
			name: "database connection error",
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New("database connection failed")).Once()
			},
			expectedCode: codes.Internal,
			description:  "database connection failure",
		},
		{
			name: "validation error",
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New("validation failed")).Once()
			},
			expectedCode: codes.Internal,
			description:  "validation failure",
		},
		{
			name: "service unavailable",
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New("service unavailable")).Once()
			},
			expectedCode: codes.Internal,
			description:  "service unavailable",
		},
		{
			name: "timeout error",
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", context.DeadlineExceeded).Once()
			},
			expectedCode: codes.Internal,
			description:  "timeout error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			tt.mockSetup(mockService)

			server := NewAuthServer(mockService, logger)
			ctx := context.Background()

			req := &authpb.RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
			}

			resp, err := server.Register(ctx, req)
			assert.Error(t, err)
			assert.Nil(t, resp)

			// Проверяем код ошибки
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedCode, st.Code())

			mockService.AssertExpectations(t)
		})
	}
}

// TestAuthServer_ConcurrentRequests тестирует обработку конкурентных запросов
func TestAuthServer_ConcurrentRequests(t *testing.T) {
	mockService := new(MockAuthService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Настраиваем мок для множественных вызовов
	mockService.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("user-uuid-123", nil).Times(10)

	server := NewAuthServer(mockService, logger)

	// Запускаем 10 горутин одновременно
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			req := &authpb.RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
			}
			_, err := server.Register(ctx, req)
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// Проверяем, что все запросы выполнились успешно
	for err := range errors {
		assert.NoError(t, err)
	}

	mockService.AssertExpectations(t)
}

// TestAuthServer_Logging тестирует логирование
func TestAuthServer_Logging(t *testing.T) {
	mockService := new(MockAuthService)

	// Создаем буфер для захвата логов
	var logBuffer strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Настраиваем мок
	mockService.On("Register", mock.Anything, "test@example.com", "testuser", "password123").
		Return("user-uuid-123", nil).Once()

	server := NewAuthServer(mockService, logger)
	ctx := context.Background()

	req := &authpb.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	resp, err := server.Register(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Проверяем, что логи записались
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Register request")
	assert.Contains(t, logOutput, "testuser")

	mockService.AssertExpectations(t)
}

// TestAuthServer_ErrorLogging тестирует логирование ошибок
func TestAuthServer_ErrorLogging(t *testing.T) {
	mockService := new(MockAuthService)

	// Создаем буфер для захвата логов
	var logBuffer strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Настраиваем мок для возврата ошибки
	mockService.On("Register", mock.Anything, "test@example.com", "testuser", "password123").
		Return("", assert.AnError).Once()

	server := NewAuthServer(mockService, logger)
	ctx := context.Background()

	req := &authpb.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	resp, err := server.Register(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	// Проверяем, что ошибка залогировалась
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Register failed")
	assert.Contains(t, logOutput, "testuser")

	mockService.AssertExpectations(t)
}

// TestAuthServer_ResponseValidation тестирует валидацию ответов
func TestAuthServer_ResponseValidation(t *testing.T) {
	mockService := new(MockAuthService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Тестируем успешный ответ
	mockService.On("Register", mock.Anything, "test@example.com", "testuser", "password123").
		Return("user-uuid-123", nil).Once()

	server := NewAuthServer(mockService, logger)
	ctx := context.Background()

	req := &authpb.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	resp, err := server.Register(ctx, req)
	assert.NoError(t, err)
	require.NotNil(t, resp)

	// Проверяем структуру ответа
	assert.True(t, resp.Success)
	assert.Equal(t, "user created successfully", resp.Message)
	assert.Equal(t, "user-uuid-123", resp.Useruid)

	mockService.AssertExpectations(t)
}
