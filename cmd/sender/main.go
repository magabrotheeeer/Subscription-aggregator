package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/magabrotheeeer/subscription-aggregator/internal/app/sender"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
)

func main() {
	cfg := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("starting sender service", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := sender.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize sender app", slog.Any("err", err))
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("sender app stopped with error", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("sender app stopped gracefully")
}
