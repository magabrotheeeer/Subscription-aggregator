package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker_GenerateAndParseToken(t *testing.T) {
	secret := "my_secret_key"
	ttl := time.Minute
	username := "testuser"

	jwtMaker := NewJWTMaker(secret, ttl)

	tokenStr, err := jwtMaker.GenerateToken(username)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	claims, err := jwtMaker.ParseToken(tokenStr)
	assert.NoError(t, err)
	assert.Equal(t, username, claims.Subject)

	assert.WithinDuration(t, time.Now().Add(ttl), claims.ExpiresAt.Time, time.Second*5)
}

func TestJWTMaker_ParseToken_InvalidSignature(t *testing.T) {
	secret := "correct_secret"
	wrongSecret := "wrong_secret"

	jwtMaker := NewJWTMaker(secret, time.Minute)
	badJWTMaker := NewJWTMaker(wrongSecret, time.Minute)

	tokenStr, err := jwtMaker.GenerateToken("user")
	assert.NoError(t, err)

	claims, err := badJWTMaker.ParseToken(tokenStr)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTMaker_ParseToken_InvalidFormat(t *testing.T) {
	jwtMaker := NewJWTMaker("secret", time.Minute)

	claims, err := jwtMaker.ParseToken("invalid.token.string")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestGetHashAndCompareHash(t *testing.T) {
    password := "mypassword"

    t.Run("success", func(t *testing.T) {
        hash, err := GetHash(password)
        require.NoError(t, err)
        require.NotEmpty(t, hash)

        err = CompareHash(hash, password)
        assert.NoError(t, err)
    })

    t.Run("fail", func(t *testing.T) {
        hash, err := GetHash(password)
        require.NoError(t, err)

        err = CompareHash(hash, "wrongpassword")
        assert.Error(t, err)
    })
}

func TestJWTMaker_TokenExpiration(t *testing.T) {
	secret := "secret"
	ttl := time.Millisecond * 50
	username := "expiringUser"

	jwtMaker := NewJWTMaker(secret, ttl)

	tokenStr, err := jwtMaker.GenerateToken(username)
	assert.NoError(t, err)

	time.Sleep(ttl + time.Millisecond * 10)

	claims, err := jwtMaker.ParseToken(tokenStr)
	assert.Error(t, err)
	assert.Nil(t, claims)
}
