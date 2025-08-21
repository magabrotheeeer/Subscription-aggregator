package register

import (
	"context"
)

type Service interface {
	Register(ctx context.Context, email, username, password string) error
}
