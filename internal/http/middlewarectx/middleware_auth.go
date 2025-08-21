// Package middlewarectx содержит HTTP middleware для обработки и проверки JWT токенов.
//
// JWTMiddleware проверяет наличие и валидность JWT токена в заголовке Authorization,
// валидирует его через gRPC-сервис, и в случае успеха добавляет в контекст
// имя пользователя и роль для дальнейшего использования в обработчиках.
//
// В случае ошибки проверки возвращает HTTP 401 Unauthorized с сообщением об ошибке.
package middlewarectx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Key тип для ключей контекста HTTP-запроса.
type Key string

const (
	// User — ключ для имени пользователя в контексте
	User Key = "username"
	// Role — ключ для роли пользователя в контексте
	Role Key = "role"
)

// Service описывает интерфейс сервиса для валидации JWT токена.
type Service interface {
	ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error)
}

// JWTMiddleware возвращает HTTP middleware, который проверяет JWT в заголовке Authorization.
//
// Если токен валиден, добавляет имя пользователя и роль в контекст запроса,
// иначе возвращает ошибку с HTTP статусом 401 Unauthorized.
func JWTMiddleware(authClient Service, log *slog.Logger) func(http.Handler) http.Handler {
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

			resp, err := authClient.ValidateToken(r.Context(), tokenStr)
			if err != nil || !resp.Valid {
				log.Error("invalid or expired token", sl.Err(err))
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("invalid or expired token"))
				return
			}
			ctx := context.WithValue(r.Context(), User, resp.Username)
			ctx = context.WithValue(ctx, Role, resp.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
