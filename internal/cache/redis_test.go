package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/magabrotheeeer/subscription-aggregator/internal/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type dummy struct {
	Field string
}

// setupRedisContainer поднимает Redis-контейнер и возвращает Cache и функцию-очистки.
func setupRedisContainer(t *testing.T) (*cache.Cache, func()) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:7.0-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort(nat.Port("6379/tcp")),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start Redis container: %v", err)
	}

	host, _ := redisC.Host(ctx)
	port, _ := redisC.MappedPort(ctx, nat.Port("6379"))

	cfg := config.RedisConnection{
		AddressRedis: host + ":" + port.Port(),
		Password:     "",
		DB:           0,
		User:         "",
		MaxRetries:   1,
		DialTimeout:  5 * time.Second,
		TimeoutRedis: 5 * time.Second,
	}
	c, err := cache.InitServer(ctx, cfg)
	if err != nil {
		t.Fatalf("InitServer failed: %v", err)
	}

	cleanup := func() {
		redisC.Terminate(ctx)
	}
	return c, cleanup
}

func TestCache_Set(t *testing.T) {
	c, cleanup := setupRedisContainer(t)
	defer cleanup()

	tests := []struct {
		name       string
		key        string
		value      any
		expiration time.Duration
		wantErr    bool
	}{
		{"valid struct", "key1", &dummy{Field: "ok"}, time.Minute, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Set(tt.key, tt.value, tt.expiration)
			require.NoError(t, err)
		})
	}
}

func TestCache_Get(t *testing.T) {
	c, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()

	data := &dummy{Field: "test"}
	testdata, _ := json.Marshal(data)
	require.NoError(t, c.Db.Set(ctx, "key1", testdata, time.Minute).Err())

	tests := []struct {
		name      string
		key       string
		value     any
		wantFound bool
		wantErr   bool
	}{
		{"key exists", "key1", &dummy{}, true, false},
		{"key not exist", "key2", &dummy{}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Get(tt.key, tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFound, got)
		})
	}
}

func TestCache_Invalidate(t *testing.T) {
	c, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()

	data := &dummy{Field: "test"}
	testdata, _ := json.Marshal(data)
	require.NoError(t, c.Db.Set(ctx, "key1", testdata, time.Minute).Err())

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"delete exists", "key1", false},
		{"delete missing", "key2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Invalidate(tt.key)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
