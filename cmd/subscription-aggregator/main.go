package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	subscriptionaggregator "github.com/magabrotheeeer/subscription-aggregator/internal/app/subscription-aggregator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
)

func main() {
	cfg := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("starting subscription-aggregator", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := subscriptionaggregator.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize app", slog.Any("err", err))
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("app stopped with error", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("subscription-aggregator stopped gracefully")
}
