package middlewarectx

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 3)

func RateLimitMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			log.Error("too many requests")
			w.WriteHeader(http.StatusTooManyRequests)
			render.JSON(w, r, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
		})
	}
}
