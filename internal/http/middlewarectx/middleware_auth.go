package middlewarectx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

type contextKey string

const UserKey contextKey = "username"

func JWTMiddleware(jwtMaker jwt.JWTMaker, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			const op = "auth.Jwtmiddleware"

			log = log.With(
				slog.String("op", op),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			if !strings.HasPrefix(authHeader, "Bearer ") {
				log.Error("missing or invalid authorization header")
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("missing or invalid authorization header"))

				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwtMaker.ParseToken(tokenStr)
			if err != nil {
				log.Error("invalid or expired token", sl.Err(err))
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("invalid or expired token"))

				return
			}
			ctx := context.WithValue(r.Context(), UserKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
