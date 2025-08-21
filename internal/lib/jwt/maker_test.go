package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker_GenerateAndParseToken_ValidCases(t *testing.T) {
	secretKey := "test_secret_key_1234567890"
	tokenTTL := 15 * time.Minute
	maker := NewJWTMaker(secretKey, tokenTTL)

	tests := []struct {
		name     string
		username string
		role     string
	}{
		{
			name:     "admin user",
			username: "admin_user",
			role:     "admin",
		},
		{
			name:     "regular user",
			username: "regular_user",
			role:     "user",
		},
		{
			name:     "user with email username",
			username: "user@domain.com",
			role:     "user",
		},
		{
			name:     "user with numbers in username",
			username: "user123",
			role:     "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := maker.GenerateToken(tt.username, tt.role)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			claims, err := maker.ParseToken(token)
			require.NoError(t, err)

			assert.Equal(t, tt.username, claims.Username)
			assert.Equal(t, tt.role, claims.Role)
			assert.WithinDuration(t, time.Now(), claims.IssuedAt.Time, time.Second)
			assert.WithinDuration(t, time.Now().Add(tokenTTL), claims.ExpiresAt.Time, time.Second)
		})
	}
}

func TestJWTMaker_ParseToken_InvalidTokens(t *testing.T) {
	secretKey := "test_secret_key_1234567890"
	tokenTTL := 15 * time.Minute
	maker := NewJWTMaker(secretKey, tokenTTL)

	validToken, err := maker.GenerateToken("testuser", "user")
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "empty token",
			token:     "",
			wantError: true,
		},
		{
			name:      "malformed token",
			token:     "invalid.token.here",
			wantError: true,
		},
		{
			name:      "expired token",
			token:     createExpiredToken(t, secretKey),
			wantError: true,
		},
		{
			name:      "wrong secret key",
			token:     createTokenWithWrongSecret(t),
			wantError: true,
		},
		{
			name:      "tampered token",
			token:     validToken + "tampered",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := maker.ParseToken(tt.token)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestJWTMaker_DifferentSecretKeys(t *testing.T) {
	maker1 := NewJWTMaker("first_secret_key", 15*time.Minute)
	maker2 := NewJWTMaker("different_secret_key", 15*time.Minute)

	token, err := maker1.GenerateToken("testuser", "admin")
	require.NoError(t, err)

	claims, err := maker2.ParseToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)

	claims, err = maker1.ParseToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
}

func createExpiredToken(t *testing.T, secretKey string) string {
	maker := NewJWTMaker(secretKey, -time.Hour)
	token, err := maker.GenerateToken("testuser", "user")
	require.NoError(t, err)
	return token
}

func createTokenWithWrongSecret(t *testing.T) string {
	wrongMaker := NewJWTMaker("wrong_secret_key", 15*time.Minute)
	token, err := wrongMaker.GenerateToken("testuser", "user")
	require.NoError(t, err)
	return token
}

func TestJWTMaker_TokenExpiration(t *testing.T) {
	secretKey := "test_secret_key"
	shortTTL := 100 * time.Millisecond
	maker := NewJWTMaker(secretKey, shortTTL)

	token, err := maker.GenerateToken("testuser", "user")
	require.NoError(t, err)

	claims, err := maker.ParseToken(token)
	require.NoError(t, err)
	assert.NotNil(t, claims)

	time.Sleep(150 * time.Millisecond)

	claims, err = maker.ParseToken(token)
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "expired")
}
