// Package mware содержит middleware для HTTP‑сервера.
// Здесь реализована проверка JWT‑токена, аутентификация пользователя
// и добавление имени пользователя в контекст запроса.
package mware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// JWTMiddleware возвращает middleware, которое проверяет JWT‑токен в заголовке Authorization.
// Логика работы:
//  1. Считывает значение заголовка Authorization.
//  2. Проверяет, что он начинается с "Bearer ".
//  3. Валидирует токен и извлекает из него Subject (имя пользователя).
//  4. Кладёт имя пользователя в контекст запроса.
//  5. Передаёт управление следующему обработчику.
func JWTMiddleware(jwtMaker auth.JWTMaker, log *slog.Logger) func(http.Handler) http.Handler {
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

			_, err := jwtMaker.ParseToken(tokenStr)
			if err != nil {
				log.Error("invalid or expired token", sl.Err(err))
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("invalid or expired token"))

				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
