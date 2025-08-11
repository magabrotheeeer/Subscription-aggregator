package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
)

type testStruct struct {
	Name string
	Age  int
}

func setupTestCache(t *testing.T) *Cache {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	t.Cleanup(func() { mr.Close() })

	cfg := config.RedisConnection{
		AddressRedis: mr.Addr(),
		Password:     "",
		DB:           0,
		User:         "",
	}

	cache, err := InitServer(context.Background(), cfg)
	require.NoError(t, err)
	return cache
}

func TestSetAndGet(t *testing.T) {
	cache := setupTestCache(t)

	expected := testStruct{Name: "Alice", Age: 30}
	err := cache.Set("user:1", expected, time.Minute)
	require.NoError(t, err)

	var actual testStruct
	found, err := cache.Get("user:1", &actual)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, expected, actual)
}

func TestGetNotFound(t *testing.T) {
	cache := setupTestCache(t)

	var out testStruct
	found, err := cache.Get("no_such_key", &out)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestInvalidate(t *testing.T) {
	cache := setupTestCache(t)

	err := cache.Set("key", "value", time.Minute)
	require.NoError(t, err)

	err = cache.Invalidate("key")
	require.NoError(t, err)

	var out string
	found, err := cache.Get("key", &out)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestGetInvalidJSON(t *testing.T) {
	cache := setupTestCache(t)

	err := cache.Db.Set(context.Background(), "bad", []byte("not-json"), time.Minute).Err()
	require.NoError(t, err)

	var out testStruct
	found, err := cache.Get("bad", &out)
	assert.False(t, found)
	assert.Error(t, err)
}

func TestInitServerInvalidAddr(t *testing.T) {
	cfg := config.RedisConnection{
		AddressRedis: "127.0.0.1:9999",
	}

	cache, err := InitServer(context.Background(), cfg)
	assert.Nil(t, cache)
	assert.Error(t, err)
}
