package client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
)

type AuthClientInterface interface {
	ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error)
	Register(ctx context.Context, email, username, password string) error
	Login(ctx context.Context, username, password string) (*authpb.LoginResponse, error)
}

type AuthClient struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c := authpb.NewAuthServiceClient(conn)
	return &AuthClient{conn: conn, client: c}, nil
}

func (a *AuthClient) Close() error {
	return a.conn.Close()
}

func (a *AuthClient) Login(ctx context.Context, username, password string) (*authpb.LoginResponse, error) {
	return a.client.Login(ctx, &authpb.LoginRequest{
		Username: username,
		Password: password,
	})
}

func (a *AuthClient) Register(ctx context.Context, email, username, password string) error {
	_, err := a.client.Register(ctx, &authpb.RegisterRequest{
		Email:    email,
		Username: username,
		Password: password,
	})
	return err
}

func (a *AuthClient) ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error) {
	return a.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{
		Token: token,
	})
}
