// @title Subscription API
// @version 1.0
// @description Сервис учёта подписок
// @host localhost:8080
// @BasePath /api/v1
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/magabrotheeeer/subscription-aggregator/cmd/docs"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/count_sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/create"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/remove"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/update"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/postgresql"
	httpSwagger "github.com/swaggo/http-swagger"
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

	router.Route("/api/v1", func(r chi.Router) {
		// Основные CRUD операции с подписками
		r.Post("/subscriptions", create.New(ctx, logger, storage))   // создать подписку
		r.Get("/subscriptions", list.New(ctx, logger, storage))      // список всех подписок
		r.Put("/subscriptions", update.New(ctx, logger, storage))    // обновить подписку
		r.Delete("/subscriptions", remove.New(ctx, logger, storage)) // удалить подписки

		// Дополнительные операции
		r.Post("/subscriptions/sum", countsum.New(ctx, logger, storage)) // сумма подписок
		r.Post("/subscriptions/filter", read.New(ctx, logger, storage))  // фильтрованный поиск
	})
	router.Get("/docs/*", httpSwagger.WrapHandler)

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
