package middlewarectx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// SubscriptionServiceInterface определяет интерфейс для работы с подписками
type SubscriptionServiceInterface interface {
	GetSubscriptionStatus(ctx context.Context, userUID string) (string, error)
}

// SubscriptionStatusMiddleware создает middleware для проверки статуса подписки пользователя.
func SubscriptionStatusMiddleware(log *slog.Logger, subService SubscriptionServiceInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userUID, ok := r.Context().Value(UserUID).(string)
			if !ok || userUID == "" {
				log.Error("user identification missing")
				w.WriteHeader(http.StatusUnauthorized)
				render.JSON(w, r, response.Error("user identification missing"))
				return
			}

			status, err := subService.GetSubscriptionStatus(r.Context(), userUID)
			if err != nil {
				log.Error("failed to get subscription status", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, response.Error("internal service error"))
				return
			}

			if status == "expired" {
				log.Error("subscription expired, access denied", sl.Err(err))
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, response.Error("subscription expired, access denied"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
