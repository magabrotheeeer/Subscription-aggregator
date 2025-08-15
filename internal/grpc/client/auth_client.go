package client

import (
	"context"
	"time"

	"google.golang.org/grpc"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

func NewAuthClient(addr string) (*AuthClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
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

func (a *AuthClient) Register(ctx context.Context, username, password string) error {
	_, err := a.client.Register(ctx, &authpb.RegisterRequest{
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
