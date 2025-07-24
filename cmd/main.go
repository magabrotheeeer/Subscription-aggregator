package main

import (
	"log/slog"
	"os"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
)

func main() {
	config := config.MustLoad()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting subscription-aggregator", slog.String("env", config.Env))
	logger.Debug("debug messages are enabled")

}