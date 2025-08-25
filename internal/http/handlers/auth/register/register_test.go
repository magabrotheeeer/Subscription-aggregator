package register

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Мок клиента с методом Register
type AuthClientMock struct {
	mock.Mock
}

func (m *AuthClientMock) Register(ctx context.Context, email, username, password string) error {
	args := m.Called(ctx, email, username, password)
	return args.Error(0)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestRegisterHandler_ServeHTTP(t *testing.T) {
	authMock := new(AuthClientMock)
	logger := newNoopLogger()

	handler := New(logger, authMock)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockErr        error
		wantStatusCode int
		wantData       map[string]any
		wantError      string
		wantStatus     string
	}{
		{
			name: "valid registration",
			requestBody: Request{
				Username: "user1",
				Password: "password123",
				Email:    "user1@example.com",
			},
			mockErr:        nil,
			wantStatusCode: http.StatusOK,
			wantData: map[string]any{
				"message":  "user created successfully",
				"username": "user1",
				"email":    "user1@example.com",
			},
			wantError:  "",
			wantStatus: "OK",
		},
		{
			name:           "invalid json body",
			requestBody:    "not a json",
			wantStatusCode: http.StatusBadRequest,
			wantData:       nil,
			wantError:      "invalid request body",
			wantStatus:     "Error",
		},
		{
			name: "validation error - missing password",
			requestBody: Request{
				Username: "user1",
				Email:    "user1@example.com",
			},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantData:       nil,
			wantError:      "field Password is a required field",
			wantStatus:     "Error",
		},
		{
			name: "registration grpc error",
			requestBody: Request{
				Username: "user1",
				Password: "password123",
				Email:    "user1@example.com",
			},
			mockErr:        errors.New("grpc error"),
			wantStatusCode: http.StatusInternalServerError,
			wantData:       nil,
			wantError:      "failed to register user",
			wantStatus:     "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock.ExpectedCalls = nil
			authMock.Calls = nil

			if tt.name == "valid registration" || tt.name == "registration grpc error" {
				authMock.On("Register", mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(tt.mockErr).Once()
			}

			var bodyBytes []byte
			var err error
			switch v := tt.requestBody.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "reqid123"))

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			var got map[string]any
			err = json.NewDecoder(rec.Body).Decode(&got)
			assert.NoError(t, err)

			assert.Equal(t, tt.wantStatus, got["status"])

			if tt.wantError != "" {
				errStr, ok := got["error"].(string)
				assert.True(t, ok)
				assert.Equal(t, tt.wantError, errStr)
			} else {
				assert.Nil(t, got["error"])
			}

			if tt.wantData != nil {
				data, ok := got["data"].(map[string]any)
				assert.True(t, ok)
				for k, v := range tt.wantData {
					assert.Equal(t, v, data[k])
				}
			} else {
				assert.Nil(t, got["data"])
			}

			authMock.AssertExpectations(t)
		})
	}
}
