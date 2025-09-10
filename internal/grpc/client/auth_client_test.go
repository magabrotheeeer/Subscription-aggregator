package client

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
)

// MockAuthServiceClient - мок для gRPC клиента
type MockAuthServiceClient struct {
	mock.Mock
}

func (m *MockAuthServiceClient) Register(ctx context.Context, req *authpb.RegisterRequest, opts ...grpc.CallOption) (*authpb.RegisterResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authpb.RegisterResponse), args.Error(1)
}

func (m *MockAuthServiceClient) Login(ctx context.Context, req *authpb.LoginRequest, opts ...grpc.CallOption) (*authpb.LoginResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authpb.LoginResponse), args.Error(1)
}

func (m *MockAuthServiceClient) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest, opts ...grpc.CallOption) (*authpb.ValidateTokenResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authpb.ValidateTokenResponse), args.Error(1)
}

// TestAuthClient_NewAuthClient тестирует создание нового клиента
func TestAuthClient_NewAuthClient(t *testing.T) {
	client, err := NewAuthClient("localhost:50051")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.conn)
	assert.NotNil(t, client.client)
}

// TestAuthClient_Close тестирует закрытие соединения
func TestAuthClient_Close(t *testing.T) {
	client, err := NewAuthClient("localhost:50051")
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Close()
	assert.NoError(t, err)
}

// TestAuthClient_Login тестирует метод Login
func TestAuthClient_Login(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		mockResponse  *authpb.LoginResponse
		mockError     error
		expectedToken string
		expectedRole  string
		expectedError bool
	}{
		{
			name:     "successful login",
			username: "testuser",
			password: "password123",
			mockResponse: &authpb.LoginResponse{
				Token:        "jwt-token-123",
				RefreshToken: "refresh-token-123",
				Role:         "user",
			},
			mockError:     nil,
			expectedToken: "jwt-token-123",
			expectedRole:  "user",
			expectedError: false,
		},
		{
			name:          "invalid credentials",
			username:      "testuser",
			password:      "wrongpassword",
			mockResponse:  nil,
			mockError:     status.Error(codes.Unauthenticated, "invalid credentials"),
			expectedError: true,
		},
		{
			name:          "empty username",
			username:      "",
			password:      "password123",
			mockResponse:  nil,
			mockError:     status.Error(codes.InvalidArgument, "username is required"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем мок клиента
			mockClient := new(MockAuthServiceClient)

			// Настраиваем ожидания для мока
			if tt.mockError != nil {
				mockClient.On("Login", mock.Anything, mock.MatchedBy(func(req *authpb.LoginRequest) bool {
					return req.Username == tt.username && req.Password == tt.password
				}), mock.Anything).Return(nil, tt.mockError).Once()
			} else {
				mockClient.On("Login", mock.Anything, mock.MatchedBy(func(req *authpb.LoginRequest) bool {
					return req.Username == tt.username && req.Password == tt.password
				}), mock.Anything).Return(tt.mockResponse, nil).Once()
			}

			// Создаем AuthClient с моком
			client := &AuthClient{
				client: mockClient,
			}

			// Выполняем тест
			ctx := context.Background()
			resp, err := client.Login(ctx, tt.username, tt.password)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedToken, resp.Token)
				assert.Equal(t, tt.expectedRole, resp.Role)
			}

			// Проверяем, что все ожидания были выполнены
			mockClient.AssertExpectations(t)
		})
	}
}

// TestAuthClient_Register тестирует метод Register
func TestAuthClient_Register(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		username      string
		password      string
		mockResponse  *authpb.RegisterResponse
		mockError     error
		expectedUID   string
		expectedError bool
	}{
		{
			name:     "successful registration",
			email:    "test@example.com",
			username: "testuser",
			password: "password123",
			mockResponse: &authpb.RegisterResponse{
				Success: true,
				Message: "user created successfully",
				Useruid: "user-uuid-123",
			},
			mockError:     nil,
			expectedUID:   "user-uuid-123",
			expectedError: false,
		},
		{
			name:          "duplicate username",
			email:         "test2@example.com",
			username:      "testuser",
			password:      "password123",
			mockResponse:  nil,
			mockError:     status.Error(codes.AlreadyExists, "username already exists"),
			expectedError: true,
		},
		{
			name:          "invalid email",
			email:         "invalid-email",
			username:      "testuser2",
			password:      "password123",
			mockResponse:  nil,
			mockError:     status.Error(codes.InvalidArgument, "invalid email format"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем мок клиента
			mockClient := new(MockAuthServiceClient)

			// Настраиваем ожидания для мока
			if tt.mockError != nil {
				mockClient.On("Register", mock.Anything, mock.MatchedBy(func(req *authpb.RegisterRequest) bool {
					return req.Email == tt.email && req.Username == tt.username && req.Password == tt.password
				}), mock.Anything).Return(nil, tt.mockError).Once()
			} else {
				mockClient.On("Register", mock.Anything, mock.MatchedBy(func(req *authpb.RegisterRequest) bool {
					return req.Email == tt.email && req.Username == tt.username && req.Password == tt.password
				}), mock.Anything).Return(tt.mockResponse, nil).Once()
			}

			// Создаем AuthClient с моком
			client := &AuthClient{
				client: mockClient,
			}

			// Выполняем тест
			ctx := context.Background()
			uid, err := client.Register(ctx, tt.email, tt.username, tt.password)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, uid)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUID, uid)
			}

			// Проверяем, что все ожидания были выполнены
			mockClient.AssertExpectations(t)
		})
	}
}

// TestAuthClient_ValidateToken тестирует метод ValidateToken
func TestAuthClient_ValidateToken(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		mockResponse  *authpb.ValidateTokenResponse
		mockError     error
		expectedValid bool
		expectedUser  string
		expectedRole  string
		expectedError bool
	}{
		{
			name:  "valid token",
			token: "valid.jwt.token",
			mockResponse: &authpb.ValidateTokenResponse{
				Valid:    true,
				Username: "testuser",
				Role:     "user",
				Useruid:  "user-uuid-123",
			},
			mockError:     nil,
			expectedValid: true,
			expectedUser:  "testuser",
			expectedRole:  "user",
			expectedError: false,
		},
		{
			name:          "invalid token",
			token:         "invalid.token",
			mockResponse:  nil,
			mockError:     status.Error(codes.Unauthenticated, "invalid token"),
			expectedError: true,
		},
		{
			name:          "expired token",
			token:         "expired.jwt.token",
			mockResponse:  nil,
			mockError:     status.Error(codes.Unauthenticated, "token expired"),
			expectedError: true,
		},
		{
			name:          "empty token",
			token:         "",
			mockResponse:  nil,
			mockError:     status.Error(codes.InvalidArgument, "token is required"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем мок клиента
			mockClient := new(MockAuthServiceClient)

			// Настраиваем ожидания для мока
			if tt.mockError != nil {
				mockClient.On("ValidateToken", mock.Anything, mock.MatchedBy(func(req *authpb.ValidateTokenRequest) bool {
					return req.Token == tt.token
				}), mock.Anything).Return(nil, tt.mockError).Once()
			} else {
				mockClient.On("ValidateToken", mock.Anything, mock.MatchedBy(func(req *authpb.ValidateTokenRequest) bool {
					return req.Token == tt.token
				}), mock.Anything).Return(tt.mockResponse, nil).Once()
			}

			// Создаем AuthClient с моком
			client := &AuthClient{
				client: mockClient,
			}

			// Выполняем тест
			ctx := context.Background()
			resp, err := client.ValidateToken(ctx, tt.token)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedValid, resp.Valid)
				assert.Equal(t, tt.expectedUser, resp.Username)
				assert.Equal(t, tt.expectedRole, resp.Role)
			}

			// Проверяем, что все ожидания были выполнены
			mockClient.AssertExpectations(t)
		})
	}
}

// TestAuthClient_ContextCancellation тестирует отмену контекста
func TestAuthClient_ContextCancellation(t *testing.T) {
	// Создаем мок клиента
	mockClient := new(MockAuthServiceClient)

	// Настраиваем ожидание для возврата ошибки контекста
	mockClient.On("Login", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, context.DeadlineExceeded).Once()

	// Создаем AuthClient с моком
	client := &AuthClient{
		client: mockClient,
		conn:   nil, // Для unit тестов соединение не нужно
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Выполняем тест
	_, err := client.Login(ctx, "testuser", "password123")

	// Проверяем, что получили ошибку отмены контекста
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestAuthClient_EdgeCases тестирует edge cases и граничные условия
func TestAuthClient_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		password    string
		email       string
		token       string
		description string
	}{
		{
			name:        "empty strings",
			username:    "",
			password:    "",
			email:       "",
			token:       "",
			description: "all empty strings",
		},
		{
			name:        "very long strings",
			username:    strings.Repeat("a", 1000),
			password:    strings.Repeat("b", 1000),
			email:       strings.Repeat("c", 1000) + "@example.com",
			token:       strings.Repeat("d", 1000),
			description: "very long input strings",
		},
		{
			name:        "special characters",
			username:    "user@#$%^&*()",
			password:    "pass!@#$%^&*()",
			email:       "test@#$%^&*().com",
			token:       "token!@#$%^&*()",
			description: "special characters in input",
		},
		{
			name:        "unicode characters",
			username:    "пользователь",
			password:    "пароль123",
			email:       "тест@пример.рф",
			token:       "токен123",
			description: "unicode characters",
		},
		{
			name:        "whitespace only",
			username:    "   ",
			password:    "\t\n\r",
			email:       "   ",
			token:       "   ",
			description: "whitespace only strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAuthServiceClient)

			// Настраиваем мок для возврата ошибки для всех edge cases
			mockClient.On("Login", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, status.Error(codes.InvalidArgument, "invalid input")).Maybe()
			mockClient.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, status.Error(codes.InvalidArgument, "invalid input")).Maybe()
			mockClient.On("ValidateToken", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, status.Error(codes.InvalidArgument, "invalid input")).Maybe()

			client := &AuthClient{
				client: mockClient,
				conn:   nil, // Для unit тестов соединение не нужно
			}
			ctx := context.Background()

			// Тестируем Login
			_, err := client.Login(ctx, tt.username, tt.password)
			assert.Error(t, err)

			// Тестируем Register
			_, err = client.Register(ctx, tt.email, tt.username, tt.password)
			assert.Error(t, err)

			// Тестируем ValidateToken
			_, err = client.ValidateToken(ctx, tt.token)
			assert.Error(t, err)

			mockClient.AssertExpectations(t)
		})
	}
}

// TestAuthClient_NetworkErrors тестирует обработку сетевых ошибок
func TestAuthClient_NetworkErrors(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		description string
	}{
		{
			name:        "connection refused",
			mockError:   status.Error(codes.Unavailable, "connection refused"),
			description: "server unavailable",
		},
		{
			name:        "timeout",
			mockError:   status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
			description: "request timeout",
		},
		{
			name:        "internal server error",
			mockError:   status.Error(codes.Internal, "internal server error"),
			description: "server internal error",
		},
		{
			name:        "unknown error",
			mockError:   status.Error(codes.Unknown, "unknown error"),
			description: "unknown server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockAuthServiceClient)

			// Настраиваем мок для возврата сетевой ошибки
			mockClient.On("Login", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.mockError).Once()

			client := &AuthClient{
				client: mockClient,
				conn:   nil, // Для unit тестов соединение не нужно
			}
			ctx := context.Background()

			_, err := client.Login(ctx, "testuser", "password123")
			assert.Error(t, err)

			// Проверяем, что ошибка правильно передается
			st, ok := status.FromError(err)
			assert.True(t, ok)
			expectedSt, _ := status.FromError(tt.mockError)
			assert.Equal(t, expectedSt.Code(), st.Code())

			mockClient.AssertExpectations(t)
		})
	}
}

// TestAuthClient_ConcurrentAccess тестирует конкурентный доступ
func TestAuthClient_ConcurrentAccess(t *testing.T) {
	mockClient := new(MockAuthServiceClient)

	// Настраиваем мок для множественных вызовов
	mockClient.On("Login", mock.Anything, mock.Anything, mock.Anything).
		Return(&authpb.LoginResponse{
			Token:        "token",
			RefreshToken: "refresh",
			Role:         "user",
		}, nil).Times(10)

	client := &AuthClient{
		client: mockClient,
		conn:   nil, // Для unit тестов соединение не нужно
	}

	// Запускаем 10 горутин одновременно
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := client.Login(ctx, "testuser", "password123")
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// Проверяем, что все запросы выполнились успешно
	for err := range errors {
		assert.NoError(t, err)
	}

	mockClient.AssertExpectations(t)
}

// TestAuthClient_ResourceCleanup тестирует правильную очистку ресурсов
func TestAuthClient_ResourceCleanup(t *testing.T) {
	// Создаем реальный клиент для тестирования очистки
	client, err := NewAuthClient("localhost:50051")
	require.NoError(t, err)
	require.NotNil(t, client)

	// Проверяем, что соединение создано
	assert.NotNil(t, client.conn)
	assert.NotNil(t, client.client)

	// Закрываем соединение
	err = client.Close()
	assert.NoError(t, err)

	// Попытка использовать закрытое соединение должна вернуть ошибку
	ctx := context.Background()
	_, err = client.Login(ctx, "testuser", "password123")
	assert.Error(t, err)
}

// TestAuthClient_InvalidAddresses тестирует создание клиента с невалидными адресами
func TestAuthClient_InvalidAddresses(t *testing.T) {
	invalidAddresses := []string{
		"",                // пустой адрес
		"invalid-address", // невалидный формат
	}

	for _, addr := range invalidAddresses {
		t.Run("address_"+addr, func(t *testing.T) {
			client, err := NewAuthClient(addr)
			// gRPC клиент может создаться успешно даже с невалидными адресами
			// Проверяем, что клиент создался (ошибка соединения будет при первом вызове)
			if err != nil {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
				// Закрываем соединение, если оно было создано
				if closeErr := client.Close(); closeErr != nil {
					t.Errorf("failed to close client: %v", closeErr)
				}
			}
		})
	}
}
