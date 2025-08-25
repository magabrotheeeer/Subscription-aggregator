package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/notification-scheduler"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting notification-scheduler", slog.String("env", cfg.Env))
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

	schedulerService := services.NewSchedulerService(db, logger)

	go schedulerService.FindExpiringSubscriptions(ctx, ch)
	select {}
}
