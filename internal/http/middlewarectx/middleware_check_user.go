package middlewarectx

import (
	"log/slog"
	"net/http"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/subscription"
)

func SubscriptionStatusMiddleware(log *slog.Logger, subService *services.SubscriptionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userUID, ok := r.Context().Value(UserUID).(string)
			if !ok || userUID == "" {
				http.Error(w, "User identification missing", http.StatusUnauthorized)
				return
			}

			status, err := subService.GetSubscriptionStatus(r.Context(), userUID)
			if err != nil {
				log.Error("failed to get subscription status", sl.Err(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if status == "expired" {
				http.Error(w, "Subscription expired, access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
