package server

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/jackc/pgx/v5/stdlib"
	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/services"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

// runMigrations запускает миграции для тестовой базы
func runMigrations(t *testing.T, connStr string) {
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer func() {
		err = db.Close()
		require.NoError(t, err)
	}()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user'
		)`,

		`CREATE TABLE IF NOT EXISTS subscriptions (
			id SERIAL PRIMARY KEY,
			service_name TEXT NOT NULL,
			price INT NOT NULL,
			username TEXT NOT NULL,
			start_date DATE NOT NULL,
			counter_months INT NOT NULL
		)`,

		`CREATE INDEX IF NOT EXISTS idx_susbcriptions_username ON subscriptions(username)`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		require.NoError(t, err, "Failed to run migration: %s", migration)
	}
}

func setupTestGRPCServer(t *testing.T) (authpb.AuthServiceClient, func()) {
	ctx := context.Background()

	// Запускаем PostgreSQL контейнер
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// Получаем connection string
	connStr, err := pgContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Запускаем миграции
	runMigrations(t, connStr)

	// Инициализируем storage
	storage, err := storage.New(connStr)
	require.NoError(t, err)

	// Инициализируем JWT maker и сервис
	jwtMaker := jwt.NewJWTMaker("test_secret_key", 24*time.Hour)
	authService := services.NewAuthService(storage, jwtMaker)

	// Запускаем gRPC сервер
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	grpcServer := grpc.NewServer()
	authServer := NewAuthServer(authService, logger)
	authpb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	// Ждем немного чтобы сервер запустился
	time.Sleep(100 * time.Millisecond)

	// Создаем клиент
	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := authpb.NewAuthServiceClient(conn)

	// Функция cleanup
	cleanup := func() {
		err = conn.Close()
		require.NoError(t, err)
		grpcServer.Stop()
		err = storage.Db.Close()
		require.NoError(t, err)
		err = pgContainer.Terminate(ctx)
		require.NoError(t, err)
	}

	return client, cleanup
}

func TestAuthServerIntegration_Register(t *testing.T) {
	client, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	tests := []struct {
		name        string
		email       string
		username    string
		password    string
		wantSuccess bool
		wantError   bool
	}{
		{
			name:        "successful registration",
			email:       "test1@example.com",
			username:    "user1",
			password:    "password123",
			wantSuccess: true,
			wantError:   false,
		},
		{
			name:        "duplicate username",
			email:       "test2@example.com",
			username:    "user1", // Дубликат
			password:    "password123",
			wantSuccess: false,
			wantError:   true,
		},
		{
			name:        "duplicate email",
			email:       "test1@example.com", // Дубликат
			username:    "user2",
			password:    "password123",
			wantSuccess: false,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Register(context.Background(), &authpb.RegisterRequest{
				Email:    tt.email,
				Username: tt.username,
				Password: tt.password,
			})

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantSuccess, resp.Success)
			}
		})
	}
}

func TestAuthServerIntegration_Login(t *testing.T) {
	client, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	// Сначала регистрируем тестового пользователя
	_, err := client.Register(ctx, &authpb.RegisterRequest{
		Email:    "login_test@example.com",
		Username: "login_user",
		Password: "correct_password",
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		username  string
		password  string
		wantToken bool
		wantError bool
		wantRole  string
	}{
		{
			name:      "successful login",
			username:  "login_user",
			password:  "correct_password",
			wantToken: true,
			wantError: false,
			wantRole:  "user",
		},
		{
			name:      "nonexistent user",
			username:  "nonexistent",
			password:  "password123",
			wantToken: false,
			wantError: true,
			wantRole:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Login(ctx, &authpb.LoginRequest{
				Username: tt.username,
				Password: tt.password,
			})

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
				assert.Equal(t, tt.wantRole, resp.Role)
				if tt.wantToken {
					assert.NotEmpty(t, resp.Token)
					assert.NotEmpty(t, resp.RefreshToken)
				}
			}
		})
	}
}

func TestAuthServerIntegration_ValidateToken(t *testing.T) {
	client, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	// Регистрируем и логинимся чтобы получить токен
	_, err := client.Register(ctx, &authpb.RegisterRequest{
		Email:    "validate_test@example.com",
		Username: "validate_user",
		Password: "password123",
	})
	require.NoError(t, err)

	loginResp, err := client.Login(ctx, &authpb.LoginRequest{
		Username: "validate_user",
		Password: "password123",
	})
	require.NoError(t, err)

	validToken := loginResp.Token

	tests := []struct {
		name      string
		token     string
		wantValid bool
		wantError bool
	}{
		{
			name:      "valid token",
			token:     validToken,
			wantValid: true,
			wantError: false,
		},
		{
			name:      "invalid token",
			token:     "invalid.token.here",
			wantValid: false,
			wantError: true,
		},
		{
			name:      "empty token",
			token:     "",
			wantValid: false,
			wantError: true,
		},
		{
			name:      "malformed token",
			token:     "invalid",
			wantValid: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.ValidateToken(ctx, &authpb.ValidateTokenRequest{
				Token: tt.token,
			})

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantValid, resp.Valid)
				if tt.wantValid {
					assert.Equal(t, "validate_user", resp.Username)
					assert.Equal(t, "user", resp.Role)
				}
			}
		})
	}
}
