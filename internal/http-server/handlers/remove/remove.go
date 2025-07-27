package remove

import (
	"context"
	"log/slog"
	"net/http"

	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type Deleter interface {
	RemoveSubscriptionEntryByUserID(ctx context.Context, entry subs.SubscriptionEntry) (int64, error)
	RemoveSubscriptionEntryByServiceName(ctx context.Context, entry subs.SubscriptionEntry) (int64, error)
}

func New(ctx context.Context, logger *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
