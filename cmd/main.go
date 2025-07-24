package main

import (
	"log/slog"
	"os"
	"fmt"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/postgresql"
)

func main() {
	config := config.MustLoad()
	fmt.Println(config)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting subscription-aggregator", slog.String("env", config.Env))
	logger.Debug("debug messages are enabled")

	storage, err := postgresql.New(config.StorageConnectionString)
	if err != nil {
		logger.Error("failed to init storage", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
		os.Exit(1)
	}
	_ = storage
}