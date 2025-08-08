package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Db *redis.Client
}

func InitServer(ctx context.Context, cfg config.RedisConnection) (*Cache, error) {
	const op = "cache.Initserver"
	db := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		Username:     cfg.User,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	})

	if err := db.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Cache{Db: db}, nil
}

func (c *Cache) Get(key string, result any) (bool, error) {
	const op = "cache.Get"
	val, err := c.Db.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	err = json.Unmarshal([]byte(val), result)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (c *Cache) Set(key string, value any, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Db.Set(context.Background(), key, jsonData, expiration).Err()
}

func (c *Cache) Invalidate(key string) error {
	return c.Db.Del(context.Background(), key).Err()
}
