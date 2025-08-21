package middlewarectx

import (
	"context"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
)

type Service interface {
	ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error)
}
