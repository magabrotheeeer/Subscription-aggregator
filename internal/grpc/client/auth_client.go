// Package client реализует gRPC клиент для взаимодействия с AuthService.
package client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
)

// AuthClient инкапсулирует gRPC клиент для взаимодействия с AuthService.
//
// Содержит gRPC-соединение и сгенерированный клиент для вызова методов аутентификации.
type AuthClient struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

// NewAuthClient создает новый AuthClient, подключаясь к указанному адресу с нешифрованным соединением.
func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c := authpb.NewAuthServiceClient(conn)
	return &AuthClient{conn: conn, client: c}, nil
}

// Close закрывает gRPC-соединение AuthClient.
func (a *AuthClient) Close() error {
	return a.conn.Close()
}

// Login вызывает gRPC метод Login с переданным именем пользователя и паролем.
func (a *AuthClient) Login(ctx context.Context, username, password string) (*authpb.LoginResponse, error) {
	return a.client.Login(ctx, &authpb.LoginRequest{
		Username: username,
		Password: password,
	})
}

// Register вызывает гRPC метод Register для регистрации пользователя с email, именем пользователя и паролем.
func (a *AuthClient) Register(ctx context.Context, email, username, password string) error {
	_, err := a.client.Register(ctx, &authpb.RegisterRequest{
		Email:    email,
		Username: username,
		Password: password,
	})
	return err
}

// ValidateToken вызывает гRPC метод ValidateToken для проверки валидности JWT токена.
func (a *AuthClient) ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error) {
	return a.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{
		Token: token,
	})
}
