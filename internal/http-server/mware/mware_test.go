package mware_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
)

type mockJWTMaker struct {
	ParseFunc func(tokenStr string) (*jwt.RegisteredClaims, error)
}

func (m *mockJWTMaker) GenerateToken(username string) (string, error) {
	return "", nil
}

func (m *mockJWTMaker) ParseToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	return m.ParseFunc(tokenStr)
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func TestJWTMiddleware(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		maker := &mockJWTMaker{
			ParseFunc: func(tokenStr string) (*jwt.RegisteredClaims, error) {
				require.Equal(t, "valid-token", tokenStr)
				return &jwt.RegisteredClaims{Subject: "testuser"}, nil
			},
		}

		// хэндлер, который проверит наличие username в контексте
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			user, ok := r.Context().Value(mware.UserKey).(string)
			require.True(t, ok)
			assert.Equal(t, "testuser", user)
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		handler := mware.JWTMiddleware(maker, makeLogger())(next)
		handler.ServeHTTP(w, req)

		assert.True(t, nextCalled, "next handler must be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		maker := &mockJWTMaker{}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called on missing header")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := mware.JWTMiddleware(maker, makeLogger())(next)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing or invalid authorization header")
	})

	t.Run("invalid Bearer prefix", func(t *testing.T) {
		maker := &mockJWTMaker{}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called on invalid prefix")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Token something")
		w := httptest.NewRecorder()

		handler := mware.JWTMiddleware(maker, makeLogger())(next)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing or invalid authorization header")
	})

	t.Run("invalid or expired token", func(t *testing.T) {
		maker := &mockJWTMaker{
			ParseFunc: func(tokenStr string) (*jwt.RegisteredClaims, error) {
				return nil, errors.New("token expired")
			},
		}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called on invalid token")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad-token")
		w := httptest.NewRecorder()

		handler := mware.JWTMiddleware(maker, makeLogger())(next)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired token")
	})
}
