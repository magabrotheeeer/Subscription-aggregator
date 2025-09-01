package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	customjwt "github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/password"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Мок для UserRepository
type UserRepoMock struct {
	mock.Mock
}

func (m *UserRepoMock) RegisterUser(ctx context.Context, user models.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *UserRepoMock) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Мок для jwt.JWTMaker с правильными типами
type JwtMakerMock struct {
	mock.Mock
}

func (m *JwtMakerMock) GenerateToken(username, role, useruid string) (string, error) {
	args := m.Called(username, role)
	return args.String(0), args.Error(1)
}

func (m *JwtMakerMock) ParseToken(token string) (*customjwt.CustomClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*customjwt.CustomClaims), args.Error(1)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		username    string
		password    string
		setupMocks  func(r *UserRepoMock)
		wantUserUID string
		wantErr     bool
		errMsg      string
	}{
		{
			name:     "successful registration",
			email:    "test@example.com",
			username: "testuser",
			password: "password123",
			setupMocks: func(r *UserRepoMock) {
				r.On("RegisterUser", mock.Anything, mock.MatchedBy(func(user models.User) bool {
					return user.Email == "test@example.com" &&
						user.Username == "testuser" &&
						user.PasswordHash != "" &&
						user.Role == "user"
				})).Return("some-uuid-string", nil).Once() // возвращаем строку UUID
			},
			wantUserUID: "some-uuid-string",
			wantErr:     false,
		},
		{
			name:     "repository error",
			email:    "test@example.com",
			username: "testuser",
			password: "password123",
			setupMocks: func(r *UserRepoMock) {
				r.On("RegisterUser", mock.Anything, mock.Anything).Return("", errors.New("db error")).Once()
			},
			wantUserUID: "",
			wantErr:     true,
			errMsg:      "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(UserRepoMock)
			jwtMock := new(JwtMakerMock)
			svc := services.NewAuthService(repo, jwtMock)

			tt.setupMocks(repo)

			got, err := svc.Register(context.Background(), tt.email, tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUserUID, got)
			}

			repo.AssertExpectations(t)
		})
	}
}
func TestAuthService_Login(t *testing.T) {
	// Правильный сырой пароль для теста
	rawPassword := "correctpassword"

	// Хэшируем пароль для мокового пользователя
	hashedPassword, err := password.GetHash(rawPassword)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	testUser := &models.User{
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: hashedPassword,
		Role:         "user",
	}

	tests := []struct {
		name        string
		username    string
		password    string
		setupMocks  func(r *UserRepoMock, j *JwtMakerMock)
		wantToken   string
		wantRefresh string
		wantRole    string
		wantErr     bool
		errMsg      string
	}{
		{
			name:     "successful login",
			username: "testuser",
			password: rawPassword, // Используем правильный сырой пароль
			setupMocks: func(r *UserRepoMock, j *JwtMakerMock) {
				r.On("GetUserByUsername", mock.Anything, "testuser").Return(testUser, nil).Once()
				j.On("GenerateToken", "testuser", "user").Return("jwt-token-123", nil).Once()
			},
			wantToken:   "jwt-token-123",
			wantRefresh: "refresh-token-placeholder",
			wantRole:    "user",
			wantErr:     false,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "password",
			setupMocks: func(r *UserRepoMock, _ *JwtMakerMock) {
				r.On("GetUserByUsername", mock.Anything, "nonexistent").Return(nil, errors.New("user not found")).Once()
			},
			wantToken:   "",
			wantRefresh: "",
			wantRole:    "",
			wantErr:     true,
			errMsg:      "user not found",
		},
		{
			name:     "wrong password",
			username: "testuser",
			password: "wrongpassword", // Неправильный пароль
			setupMocks: func(r *UserRepoMock, _ *JwtMakerMock) {
				r.On("GetUserByUsername", mock.Anything, "testuser").Return(testUser, nil).Once()
			},
			wantToken:   "",
			wantRefresh: "",
			wantRole:    "",
			wantErr:     true,
			errMsg:      "invalid credentials",
		},
		{
			name:     "token generation error",
			username: "testuser",
			password: rawPassword, // Правильный пароль
			setupMocks: func(r *UserRepoMock, j *JwtMakerMock) {
				r.On("GetUserByUsername", mock.Anything, "testuser").Return(testUser, nil).Once()
				j.On("GenerateToken", "testuser", "user").Return("", errors.New("token error")).Once()
			},
			wantToken:   "",
			wantRefresh: "",
			wantRole:    "",
			wantErr:     true,
			errMsg:      "token error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(UserRepoMock)
			jwtMock := new(JwtMakerMock)
			svc := services.NewAuthService(repo, jwtMock)

			tt.setupMocks(repo, jwtMock)

			token, refresh, role, err := svc.Login(context.Background(), tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
				assert.Equal(t, tt.wantRefresh, refresh)
				assert.Equal(t, tt.wantRole, role)
			}

			repo.AssertExpectations(t)
			jwtMock.AssertExpectations(t)
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	// Используем правильный тип CustomClaims из вашего пакета jwt
	validClaims := &customjwt.CustomClaims{
		Username: "testuser",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	tests := []struct {
		name       string
		token      string
		setupMocks func(j *JwtMakerMock)
		wantUser   *models.User
		wantRole   string
		wantValid  bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:  "valid token",
			token: "valid-token",
			setupMocks: func(j *JwtMakerMock) {
				j.On("ParseToken", "valid-token").Return(validClaims, nil).Once()
			},
			wantUser: &models.User{
				Username: "testuser",
				Role:     "user",
			},
			wantRole:  "user",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			setupMocks: func(j *JwtMakerMock) {
				j.On("ParseToken", "invalid-token").Return(nil, errors.New("invalid token")).Once()
			},
			wantUser:  nil,
			wantRole:  "",
			wantValid: false,
			wantErr:   true,
			errMsg:    "invalid token",
		},
		{
			name:  "expired token",
			token: "expired-token",
			setupMocks: func(j *JwtMakerMock) {
				j.On("ParseToken", "expired-token").Return(nil, errors.New("token expired")).Once()
			},
			wantUser:  nil,
			wantRole:  "",
			wantValid: false,
			wantErr:   true,
			errMsg:    "token expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(UserRepoMock)
			jwtMock := new(JwtMakerMock)
			svc := services.NewAuthService(repo, jwtMock)

			tt.setupMocks(jwtMock)

			user, role, valid, err := svc.ValidateToken(context.Background(), tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantUser, user)
			assert.Equal(t, tt.wantRole, role)
			assert.Equal(t, tt.wantValid, valid)

			jwtMock.AssertExpectations(t)
		})
	}
}
