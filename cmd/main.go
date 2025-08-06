// @title Subscription API
// @version 1.0
// @description Сервис учёта подписок
// @host localhost:8080
// @BasePath /api/v1
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/magabrotheeeer/subscription-aggregator/docs"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/count_sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/create"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/login"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/register"
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

	jwtSecretKey := os.Getenv("JWT_SECRET")
	if jwtSecretKey == "" {
		logger.Error("JWT_SECRET is not set")
		os.Exit(1)
	}
	tokenTTL := time.Hour * 24
	jwtMaker := auth.NewJWTMaker(jwtSecretKey, tokenTTL)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", register.New(ctx, logger, storage))
		r.Post("/login", login.New(ctx, logger, storage, jwtMaker))
	})

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.JWTMiddleware(jwtMaker, logger))
		// Основные CRUD операции с подписками
		r.Post("/subscriptions/", create.New(ctx, logger, storage))
		r.Get("/subscriptions/{id}", read.New(ctx, logger, storage))
		r.Put("/subscriptions/{id}", update.New(ctx, logger, storage))
		r.Delete("/subscriptions/{id}", remove.New(ctx, logger, storage))

		// Дополнительные операции
		r.Get("/subscriptions/list", list.New(ctx, logger, storage))
		r.Post("/subscriptions/sum/{id}", countsum.New(ctx, logger, storage))
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

	serverError := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverError:
		logger.Error("server error:", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
	case <-stop:
		logger.Info("shutting down gracefully...")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server shutdown error:", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
		}
		// Закрытие соединения с БД (если не defer выше)
		if err := storage.Db.Close(ctx); err != nil {
			logger.Error("DB close error:", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
		}
		logger.Info("server exited properly")
	}
}
