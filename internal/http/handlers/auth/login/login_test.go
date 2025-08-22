package login

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

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
)

type AuthClientMock struct {
	mock.Mock
}

func (m *AuthClientMock) Login(ctx context.Context, username, password string) (*authpb.LoginResponse, error) {
	args := m.Called(ctx, username, password)
	resp, _ := args.Get(0).(*authpb.LoginResponse)
	return resp, args.Error(1)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestLoginHandler_ServeHTTP(t *testing.T) {
	authMock := new(AuthClientMock)
	logger := newNoopLogger()

	handler := New(logger, authMock)

	token := "tok"
	refreshToken := "ref"
	role := "user"

	tests := []struct {
		name           string
		requestBody    interface{}
		mockResp       *authpb.LoginResponse
		mockErr        error
		wantStatusCode int
		wantData       map[string]any
		wantError      string
		wantStatus     string
	}{
		{
			name:        "valid login",
			requestBody: Request{Username: "user1", Password: "password123"},
			mockResp: &authpb.LoginResponse{
				Token:        token,
				RefreshToken: refreshToken,
				Role:         role,
			},
			mockErr:        nil,
			wantStatusCode: http.StatusOK,
			wantData: map[string]any{
				"token":         token,
				"refresh_token": refreshToken,
				"role":          role,
				"username":      "user1",
			},
			wantError:  "",
			wantStatus: "OK",
		},
		{
			name:           "invalid json body",
			requestBody:    "not a json",
			wantStatusCode: http.StatusOK,
			wantData:       nil,
			wantError:      "invalid request body",
			wantStatus:     "Error",
		},
		{
			name:           "validation error - missing password",
			requestBody:    Request{Username: "user1"},
			wantStatusCode: http.StatusOK,
			wantData:       nil,
			wantError:      "field Password is a required field",
			wantStatus:     "Error",
		},
		{
			name:           "login grpc error",
			requestBody:    Request{Username: "user1", Password: "password123"},
			mockResp:       nil,
			mockErr:        errors.New("grpc error"),
			wantStatusCode: http.StatusInternalServerError,
			wantData:       nil,
			wantError:      "invalid credentials",
			wantStatus:     "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock.ExpectedCalls = nil
			authMock.Calls = nil

			if tt.mockResp != nil || tt.mockErr != nil {
				authMock.On("Login", mock.Anything, tt.requestBody.(Request).Username, tt.requestBody.(Request).Password).
					Return(tt.mockResp, tt.mockErr).Once()
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

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyBytes))
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

			if tt.mockResp != nil || tt.mockErr != nil {
				authMock.AssertExpectations(t)
			}
		})
	}
}
