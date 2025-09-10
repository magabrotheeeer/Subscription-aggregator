// Package main содержит точку входа для сервиса аутентификации.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/magabrotheeeer/subscription-aggregator/internal/app/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
)

func main() {
	cfg := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("starting auth-service", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := auth.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize auth app", slog.Any("err", err))
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("auth app stopped with error", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("auth app stopped gracefully")
}
