// Package cache предоставляет функции и структуру для работы с Redis
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/redis/go-redis/v9"
)

// Cache предоставляет подключение к бд
type Cache struct {
	Db *redis.Client
}

// InitServer создает объект cache с настройками из config/config.yaml
func InitServer(ctx context.Context, cfg config.RedisConnection) (*Cache, error) {
	const op = "cache.Initserver"
	db := redis.NewClient(&redis.Options{
		Addr:         cfg.AddressRedis,
		Password:     cfg.Password,
		DB:           cfg.DB,
		Username:     cfg.User,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.TimeoutRedis,
		WriteTimeout: cfg.TimeoutRedis,
	})

	if err := db.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Cache{Db: db}, nil
}

// Get проверяет наличие значения по ключу key и записывает это же значение результат в result
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

// Set добавляет новое значение value по ключу key
// expiration - время хранение в redis
func (c *Cache) Set(key string, value any, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Db.Set(context.Background(), key, jsonData, expiration).Err()
}

// Invalidate удаляет значение по ключу key
func (c *Cache) Invalidate(key string) error {
	return c.Db.Del(context.Background(), key).Err()
}
