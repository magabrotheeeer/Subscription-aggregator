package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/postgresql"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/create"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/remove"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/update"
)

func main() {
	ctx := context.Background()
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

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/subs-aggregator/create/", create.New(ctx, logger, storage))
	router.Get("/subs-aggregator/read/", read.New(ctx, logger, storage))
	router.Put("/subs-aggregator/update/", update.New(ctx, logger, storage))
	router.Delete("/subs-aggregator/remove/", remove.New(ctx, logger, storage))
	router.Get("/subs-aggregator/list/", list.New(ctx, logger, storage))

	logger.Info("starting the server", slog.String("address", config.Address))

	srv := &http.Server{
		Addr:         config.Address,
		Handler:      router,
		ReadTimeout:  config.HTTPServer.Timeout,
		WriteTimeout: config.HTTPServer.Timeout,
		IdleTimeout:  config.HTTPServer.IdleTimeout,
	}

	if err = srv.ListenAndServe(); err != nil {
		logger.Error("failed to start the server", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
		os.Exit(1)
	}
	logger.Error("server stopped")

}
