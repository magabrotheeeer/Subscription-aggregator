package login

import (
	"context"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
)

type Service interface {
	Login(ctx context.Context, username, password string) (*authpb.LoginResponse, error)
}
