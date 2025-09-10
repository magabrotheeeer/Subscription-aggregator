// Package main Subscription Aggregator API
//
// @title           Subscription Aggregator API
// @version         1.0
// @description     API для управления подписками пользователей
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
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
