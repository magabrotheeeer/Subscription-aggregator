package login_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/login"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type mockUserGetter struct {
	GetFunc func(ctx context.Context, username string) (*models.User, error)
}

func (m *mockUserGetter) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return m.GetFunc(ctx, username)
}

type mockJWTMaker struct {
	GenerateFunc func(username string) (string, error)
	ParseFunc    func(tokenStr string) (*jwt.RegisteredClaims, error)
}

func (m *mockJWTMaker) GenerateToken(username string) (string, error) {
	return m.GenerateFunc(username)
}

func (m *mockJWTMaker) ParseToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	return m.ParseFunc(tokenStr)
}

// discard logger
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func TestLoginHandler(t *testing.T) {
	ctx := context.Background()

	hash, _ := auth.GetHash("password123")

	t.Run("success", func(t *testing.T) {
		// тело запроса
		body, _ := json.Marshal(login.Request{
			Username: "validuser",
			Password: "password123",
		})

		userGetter := &mockUserGetter{
			GetFunc: func(ctx context.Context, username string) (*models.User, error) {
				return &models.User{
					Username:     "validuser",
					PasswordHash: hash,
				}, nil
			},
		}
		jwtMaker := &mockJWTMaker{
			GenerateFunc: func(username string) (string, error) {
				require.Equal(t, "validuser", username)
				return "jwt-token-123", nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := login.New(ctx, makeLogger(), userGetter, jwtMaker)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, "jwt-token-123", resp.Data.(map[string]any)["token"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		userGetter := &mockUserGetter{
			GetFunc: func(ctx context.Context, username string) (*models.User, error) {
				t.Fatal("GetUserByUsername should not be called")
				return nil, nil
			},
		}
		jwtMaker := &mockJWTMaker{}

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{bad json")))
		w := httptest.NewRecorder()

		handler := login.New(ctx, makeLogger(), userGetter, jwtMaker)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode request")
	})

	t.Run("validation error", func(t *testing.T) {
		body, _ := json.Marshal(login.Request{
			Username: "",
			Password: "",
		})
		userGetter := &mockUserGetter{
			GetFunc: func(ctx context.Context, username string) (*models.User, error) {
				t.Fatal("GetUserByUsername should not be called")
				return nil, nil
			},
		}

		jwtMaker := &mockJWTMaker{}

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := login.New(ctx, makeLogger(), userGetter, jwtMaker)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "is a required field")
	})

	t.Run("user not found", func(t *testing.T) {
		body, _ := json.Marshal(login.Request{
			Username: "validuser",
			Password: "password123",
		})

		userGetter := &mockUserGetter{
			GetFunc: func(ctx context.Context, username string) (*models.User, error) {
				return nil, errors.New("not found")
			},
		}
		jwtMaker := &mockJWTMaker{}

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := login.New(ctx, makeLogger(), userGetter, jwtMaker)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "incorrect user or password")
	})

	t.Run("wrong password", func(t *testing.T) {
		body, _ := json.Marshal(login.Request{
			Username: "validuser",
			Password: "wrongpass",
		})

		userGetter := &mockUserGetter{
			GetFunc: func(ctx context.Context, username string) (*models.User, error) {
				return &models.User{
					Username:     "validuser",
					PasswordHash: hash,
				}, nil
			},
		}
		jwtMaker := &mockJWTMaker{}

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := login.New(ctx, makeLogger(), userGetter, jwtMaker)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "incorrect user or password")
	})

	t.Run("token generation error", func(t *testing.T) {
		body, _ := json.Marshal(login.Request{
			Username: "validuser",
			Password: "password123",
		})

		userGetter := &mockUserGetter{
			GetFunc: func(ctx context.Context, username string) (*models.User, error) {
				return &models.User{Username: "validuser", PasswordHash: hash}, nil
			},
		}
		jwtMaker := &mockJWTMaker{
			GenerateFunc: func(username string) (string, error) {
				return "", errors.New("jwt error")
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := login.New(ctx, makeLogger(), userGetter, jwtMaker)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "could not generate token")
	})
}
