package subscriptionaggregator

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/magabrotheeeer/subscription-aggregator/internal/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/client"
	"github.com/magabrotheeeer/subscription-aggregator/internal/paymentprovider"

	"github.com/magabrotheeeer/subscription-aggregator/internal/migrations"
	subsaggregatorservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/subscription"
	paymentservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/payment"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

type App struct {
	server *http.Server
	logger *slog.Logger
	db     *storage.Storage
	cache  cache.Cache
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	db, err := storage.New(cfg.StorageConnectionString)
	if err != nil {
		return nil, err
	}
	if err = migrations.Run(db.Db, "./migrations"); err != nil {
		return nil, err
	}

	cacheRedis, err := cache.InitServer(ctx, cfg.RedisConnection)
	if err != nil {
		return nil, err
	}

	authClient, err := client.NewAuthClient(cfg.GRPCAuthAddress)
	if err != nil {
		return nil, err
	}

	providerService := paymentprovider.NewClient("заглушка", "заглушка")
	paymentService := paymentservice.New(db, logger)
	subscriptionService := subsaggregatorservice.NewSubscriptionService(db, cacheRedis, logger)

	router := chi.NewRouter()

	RegisterRoutes(router, logger, subscriptionService, authClient, providerService, paymentService)

	router.Get("/docs/*", httpSwagger.WrapHandler)

	srv := &http.Server{
		Addr:         cfg.AddressHTTP,
		Handler:      router,
		ReadTimeout:  cfg.TimeoutHTTP,
		WriteTimeout: cfg.TimeoutHTTP,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &App{
		server: srv,
		logger: logger,
		db:     db,
		cache:  *cacheRedis,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server starting on", slog.String("address", a.server.Addr))
		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
		} else {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		a.logger.Info("shutting down HTTP server gracefully")
		err := a.server.Shutdown(timeoutCtx)
		a.db.Db.Close()
		return err
	}
}
