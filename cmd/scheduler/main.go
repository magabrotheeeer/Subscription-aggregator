package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/magabrotheeeer/subscription-aggregator/internal/app/scheduler"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
)

func main() {
	cfg := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("starting scheduler", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := scheduler.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize scheduler app", slog.Any("err", err))
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("scheduler app stopped with error", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("scheduler app stopped gracefully")
}
