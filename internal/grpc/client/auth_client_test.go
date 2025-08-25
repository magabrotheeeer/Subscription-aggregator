package client

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/server"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/services"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func runMigrations(t *testing.T, connStr string) {
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`,
		`
        CREATE TABLE IF NOT EXISTS users (
            uid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'user'
        );	
		`,
		`
        CREATE TABLE IF NOT EXISTS subscriptions (
            id SERIAL PRIMARY KEY,
            service_name TEXT NOT NULL,
            price INT NOT NULL,
            username TEXT NOT NULL,
            start_date DATE NOT NULL,
            counter_months INT NOT NULL
        );
		`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_username ON subscriptions(username);`, // исправил опечатку idx_susbcriptions
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		require.NoErrorf(t, err, "Failed to run migration: %s", migration)
	}
}

func TestAuthGRPCIntegration(t *testing.T) {
	ctx := context.Background()

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
	defer func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	}()

	connStr, err := pgContainer.ConnectionString(ctx)
	require.NoError(t, err)

	runMigrations(t, connStr)

	storage, err := storage.New(connStr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, storage.Db.Close())
	}()

	jwtMaker := jwt.NewJWTMaker("test_secret_key", 24*time.Hour)
	authService := services.NewAuthService(storage, jwtMaker)

	grpcServer, addr := startGRPCServer(t, authService)
	defer grpcServer.Stop()

	client, err := NewAuthClient(addr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	t.Run("Register and Login", func(t *testing.T) {
		err := client.Register(ctx, "test@example.com", "testuser", "password123")
		require.NoError(t, err)

		loginResp, err := client.Login(ctx, "testuser", "password123")
		require.NoError(t, err)

		assert.NotEmpty(t, loginResp.Token)
		assert.NotEmpty(t, loginResp.RefreshToken)
		assert.Equal(t, "user", loginResp.Role)
	})

	t.Run("Validate Token", func(t *testing.T) {
		loginResp, err := client.Login(ctx, "testuser", "password123")
		require.NoError(t, err)

		validateResp, err := client.ValidateToken(ctx, loginResp.Token)
		require.NoError(t, err)

		assert.True(t, validateResp.Valid)
		assert.Equal(t, "testuser", validateResp.Username)
		assert.Equal(t, "user", validateResp.Role)
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		_, err := client.Login(ctx, "testuser", "wrongpassword")
		assert.Error(t, err)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		_, err := client.ValidateToken(ctx, "invalid.token.here")
		assert.Error(t, err)
	})
}

func startGRPCServer(t *testing.T, authService *services.AuthService) (*grpc.Server, string) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	authServer := server.NewAuthServer(authService, logger)
	authpb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			t.Logf("gRPC server error: %v", serveErr)
		}
	}()

	// Ждем немного для надёжного запуска сервера
	time.Sleep(100 * time.Millisecond)

	return grpcServer, lis.Addr().String()
}
