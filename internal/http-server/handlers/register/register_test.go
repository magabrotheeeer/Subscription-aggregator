package register_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/register"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
)

type mockRegistration struct {
	RegisterFunc func(ctx context.Context, username, passwordHash string) (int, error)
}

func (m *mockRegistration) RegisterUser(ctx context.Context, username, passwordHash string) (int, error) {
	return m.RegisterFunc(ctx, username, passwordHash)
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func TestRegisterHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		body, _ := json.Marshal(register.Request{
			Username: "testuser",
			Password: "password123",
		})

		reg := &mockRegistration{
			RegisterFunc: func(ctx context.Context, username, passwordHash string) (int, error) {
				require.Equal(t, "testuser", username)
				// проверяем, что вернулся bcrypt-хэш
				require.NotEqual(t, "password123", passwordHash)
				return 100, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := register.New(ctx, makeLogger(), reg)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, "testuser", resp.Data.(map[string]any)["username"])
		assert.Equal(t, float64(100), resp.Data.(map[string]any)["id"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		reg := &mockRegistration{
			RegisterFunc: func(ctx context.Context, u, p string) (int, error) {
				t.Fatal("RegisterUser should not be called")
				return 0, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("{bad json")))
		w := httptest.NewRecorder()

		handler := register.New(ctx, makeLogger(), reg)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode request")
	})

	t.Run("validation error", func(t *testing.T) {
		body, _ := json.Marshal(register.Request{
			Username: "usr", // слишком короткое имя
			Password: "",    // пустой пароль
		})
		reg := &mockRegistration{
			RegisterFunc: func(ctx context.Context, u, p string) (int, error) {
				t.Fatal("RegisterUser should not be called")
				return 0, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := register.New(ctx, makeLogger(), reg)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "is not a valid") // min или required
	})

	t.Run("RegisterUser error", func(t *testing.T) {
		body, _ := json.Marshal(register.Request{
			Username: "testuser",
			Password: "password123",
		})
		reg := &mockRegistration{
			RegisterFunc: func(ctx context.Context, username, passwordHash string) (int, error) {
				return 0, errors.New("db error")
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := register.New(ctx, makeLogger(), reg)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to register new user")
	})
}
