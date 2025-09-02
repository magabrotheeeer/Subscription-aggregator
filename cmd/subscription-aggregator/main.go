// @title Subscription Aggregator API
// @version 1.0
// @description REST API сервис для работы с подписками (создание, удаление, обновление, получение, суммарный расчёт).
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// Package main запускает HTTP-сервер Subscription Aggregator.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/magabrotheeeer/subscription-aggregator/docs"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/magabrotheeeer/subscription-aggregator/internal/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/client"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/auth/login"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/auth/register"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/create"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/remove"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/update"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/migrations"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/subscription"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func main() {
	ctx := context.Background()
	config := config.MustLoad()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting subscription-aggregator", slog.String("env", config.Env))
	logger.Debug("debug messages are enabled")

	db, err := storage.New(config.StorageConnectionString)
	if err != nil {
		logger.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	if err = migrations.Run(db.Db, "./migrations"); err != nil {
		logger.Error("failed to run migrations", slog.Any("err", err))
		os.Exit(1)
	}
	cache, err := cache.InitServer(ctx, config.RedisConnection)
	if err != nil {
		logger.Error("failed to init cache redis", sl.Err(err))
		os.Exit(1)
	}

	authClient, err := client.NewAuthClient(config.GRPCAuthAddress)
	if err != nil {
		logger.Error("failed to connect auth grpc", sl.Err(err))
		os.Exit(1)
	}

	defer func() {
		_ = authClient.Close()
	}()

	subscriptionService := services.NewSubscriptionService(db, cache, logger)

	//TODO
	// client := paymentprovider.NewClient( /*TODO*/ )
	// paymentMethodsService := servicespm.NewPaymentMethodsService(db, logger)

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recoverer,
		middleware.URLFormat,
	)

	// Регистрация роутов API версии v1
	router.Route("/api/v1", func(r chi.Router) {
		// Открытые эндпоинты регистрации и логина
		r.Post("/register", register.New(logger, authClient, subscriptionService).ServeHTTP)
		r.Post("/login", login.New(logger, authClient).ServeHTTP)

		// Группа роутов с JWT Middleware (требуют авторизации)
		r.Group(func(r chi.Router) {
			r.Use(middlewarectx.JWTMiddleware(authClient, logger))

			r.Post("/subscriptions", create.New(logger, subscriptionService).ServeHTTP)
			r.Get("/subscriptions/{id}", read.New(logger, subscriptionService).ServeHTTP)
			r.Delete("/subscriptions/{id}", remove.New(logger, subscriptionService).ServeHTTP)
			r.Put("/subscriptions/{id}", update.New(logger, subscriptionService).ServeHTTP)

			r.Get("/subscriptions/list", list.New(logger, subscriptionService).ServeHTTP)
			r.Post("/subscriptions/sum", sum.New(logger, subscriptionService).ServeHTTP)

			//TODO
			// r.Post("/paymentmethods", paymentcreate.New(logger, client, paymentMethodsService).ServeHTTP)
		})
	})

	router.Get("/docs/*", httpSwagger.WrapHandler)

	logger.Info("starting the server", slog.String("address", config.AddressHTTP))

	srv := &http.Server{
		Addr:         config.AddressHTTP,
		Handler:      router,
		ReadTimeout:  config.TimeoutHTTP,
		WriteTimeout: config.TimeoutHTTP,
		IdleTimeout:  config.IdleTimeout,
	}

	serverError := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverError:
		logger.Error("server error", sl.Err(err))
	case <-stop:
		logger.Info("shutting down gracefully...")
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctxTimeout)
		_ = db.Db.Close()
	}
}
