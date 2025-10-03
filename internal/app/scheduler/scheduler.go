// Package scheduler содержит логику планировщика для обработки подписок.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	schedulerservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/scheduler"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/repository"
	"github.com/streadway/amqp"
)

// App представляет приложение планировщика.
type App struct {
	schedulerService *schedulerservice.SchedulerService
	conn             *amqp.Connection
	ch               *amqp.Channel
	logger           *slog.Logger
}

func waitForDB(db *repository.Storage) error {
	for range 10 {
		err := repository.CheckDatabaseReady(db)
		if err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("database not ready after retries")
}

// New создает новый экземпляр приложения планировщика.
func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	conn, err := rabbitmq.Connect(cfg.RabbitMQURL, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay)
	if err != nil {
		return nil, fmt.Errorf("failed to connect RabbitMQ: %w", err)
	}

	queues := rabbitmq.GetNotificationQueues()
	ch, err := rabbitmq.SetupChannel(conn, queues)
	if err != nil {
		closeResources(nil, conn, logger)
		return nil, fmt.Errorf("failed to setup RabbitMQ channel: %w", err)
	}

	db, err := repository.New(cfg.StorageConnectionString)
	if err != nil {
		closeResources(ch, conn, logger)
		return nil, fmt.Errorf("failed to connect storage: %w", err)
	}

	if err := waitForDB(db); err != nil {
		closeResources(ch, conn, logger)
		return nil, err
	}

	cacheRedis, err := cache.InitServer(ctx, cfg.RedisConnection)
	if err != nil {
		closeResources(ch, conn, logger)
		return nil, fmt.Errorf("cache not initialized: %w", err)
	}

	schedulerService := schedulerservice.NewSchedulerService(db, cacheRedis, logger)

	return &App{
		schedulerService: schedulerService,
		conn:             conn,
		ch:               ch,
		logger:           logger,
	}, nil
}

func closeResources(ch *amqp.Channel, conn *amqp.Connection, logger *slog.Logger) {
	if ch != nil {
		if err := ch.Close(); err != nil {
			logger.Error("failed to close channel", "error", err)
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil {
			logger.Error("failed to close connection", "error", err)
		}
	}
}

// Run запускает планировщик.
func (a *App) Run(ctx context.Context) error {
	go a.schedulerService.FindExpiringSubscriptionsDueTomorrow(ctx, a.ch)
	go a.schedulerService.FindExpiringSubscriptionsDueToday(ctx, a.ch)

	<-ctx.Done()

	a.logger.Info("shutting down scheduler service")

	if err := a.ch.Close(); err != nil {
		a.logger.Error("failed to close channel", slog.Any("err", err))
	}

	if err := a.conn.Close(); err != nil {
		a.logger.Error("failed to close connection", slog.Any("err", err))
	}

	return nil
}
