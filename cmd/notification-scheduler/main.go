package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
)

func main() {
	cfg := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting notification-scheduler", slog.String("env", cfg.Env))
	conn, err := rabbitmq.Connect(*cfg)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("succes to connect to RabbitMQ:", slog.String("URL", cfg.RabbitMQURL))
	defer func() {
		_ = conn.Close()
	}()

	ch, err := rabbitmq.SetupChannel(conn)
	if err != nil {
		logger.Error("failed to setup RabbitMQ channel", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("success to setup RabbitMQ channel")
	defer func() {
		_ = ch.Close()
	}()
	// TODO: подключение к Postgresql

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		logger.Info("запускаем поиск подписок и публикацию сообщений")
		// TODO: поиска в бд
		// TODO: публикация сообщений в rabbitmq
	}
}
