package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/rabbitmq"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/scheduler"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func waitForDB(db *storage.Storage) error {
	for range 10 {
		err := storage.CheckDatabaseReady(db)
		if err == nil {
			return nil // готово
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("database not ready after retries")
}

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting scheduler", slog.String("env", cfg.Env))
	conn, err := rabbitmq.Connect(cfg.RabbitMQURL, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("succes to connect to RabbitMQ:", slog.String("URL", cfg.RabbitMQURL))
	defer func() {
		_ = conn.Close()
	}()

	queues := rabbitmq.GetNotificationQueues()
	ch, err := rabbitmq.SetupChannel(conn, queues)
	if err != nil {
		logger.Error("failed to setup RabbitMQ channel", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("success to setup RabbitMQ channel")
	defer func() {
		_ = ch.Close()
	}()
	db, err := storage.New(cfg.StorageConnectionString)
	if err != nil {
		logger.Error("failed to connect to storage", sl.Err(err))
		os.Exit(1)
	}
	defer func() {
		_ = db.Db.Close()
	}()
	err = waitForDB(db)
	if err != nil {
		logger.Error("Database is not ready:", sl.Err(err))
	}

	schedulerService := services.NewSchedulerService(db, logger)

	go schedulerService.FindExpiringSubscriptionsDueTomorrow(ctx, ch)
	select {}
}
