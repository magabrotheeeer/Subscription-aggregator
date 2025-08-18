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
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/magabrotheeeer/subscription-aggregator/internal/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/client"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/auth/login"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/auth/register"
	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/count_sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/create"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/remove"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/subscription/update"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/services"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func main() {
	ctx := context.Background()
	config := config.MustLoad()
	fmt.Println(config.String())

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting subscription-aggregator", slog.String("env", config.Env))
	logger.Debug("debug messages are enabled")

	storage, err := storage.New(config.StorageConnectionString)
	if err != nil {
		logger.Error("failed to init storage", slog.Any("err", err))
		os.Exit(1)
	}
	cache, err := cache.InitServer(ctx, config.RedisConnection)
	if err != nil {
		logger.Error("failed to init cache redis", slog.Any("err", err))
		os.Exit(1)
	}

	authClient, err := client.NewAuthClient(config.GRPCAuthAddress)
	if err != nil {
		logger.Error("failed to connect auth grpc", slog.Any("err", err))
		os.Exit(1)
	}
	defer authClient.Close()

	jwtMaker := jwt.NewJWTMaker(config.JWTSecretKey, config.TokenTTL)

	subscriptionService := services.NewSubscriptionService(storage, cache, logger)

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recoverer,
		middleware.URLFormat,
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", register.New(logger, authClient).ServeHTTP)
		r.Post("/login", login.New(logger, authClient).ServeHTTP)

		r.Group(func(r chi.Router) {
			r.Use(middlewarectx.JWTMiddleware(jwtMaker, logger))

			r.Post("/subscriptions", create.New(logger, subscriptionService).ServeHTTP)
			r.Get("/subscriptions/{id}", read.New(logger, subscriptionService).ServeHTTP)
			r.Delete("/subscriptions/{id}", remove.New(logger, subscriptionService).ServeHTTP)
			r.Put("/subscriptions/{id}", update.New(logger, subscriptionService).ServeHTTP)

			r.Get("/subscriptions/list", list.New(logger, subscriptionService).ServeHTTP)
			r.Post("/subscriptions/sum", countsum.New(logger, subscriptionService).ServeHTTP)
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
		logger.Error("server error", slog.Any("err", err))
	case <-stop:
		logger.Info("shutting down gracefully...")
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctxTimeout)
		storage.Db.Close(ctxTimeout)
	}
}
