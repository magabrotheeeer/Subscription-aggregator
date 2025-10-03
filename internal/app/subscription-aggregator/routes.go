// Package subscriptionaggregator предоставляет маршруты для основного приложения.
package subscriptionaggregator

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/auth/login"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/auth/register"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentcreate"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentlist"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/create"

	//	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/health"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/remove"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/subscription/update"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/middlewarectx"
	paymentservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/payment"
	senderservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/sender"
	subservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/subscription"
	"github.com/magabrotheeeer/subscription-aggregator/internal/yookassa"

	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/client"
)

// RegisterRoutes регистрирует все маршруты приложения.
func RegisterRoutes(r chi.Router, logger *slog.Logger,
	subscriptionService *subservice.SubscriptionService,
	authClient *client.AuthClient,
	providerClient *yookassa.Client,
	paymentService *paymentservice.Service,
	senderService *senderservice.SenderService) {
	// Глобальные middleware
	r.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recoverer,
		middleware.URLFormat,
	)

	r.Route("/api/v1", func(r chi.Router) {
		// Открытые конечные точки
		r.Post("/register", register.New(logger, authClient, subscriptionService).ServeHTTP)
		r.Post("/login", login.New(logger, authClient).ServeHTTP)

		// Группа с JWT аутентификацией
		r.Group(func(r chi.Router) {
			r.Use(middlewarectx.JWTMiddleware(logger, authClient))
			r.Use(middlewarectx.SubscriptionStatusMiddleware(logger, subscriptionService))
			r.Use(middlewarectx.RateLimitMiddleware(logger))
			r.Post("/subscriptions", create.New(logger, subscriptionService).ServeHTTP)
			r.Get("/subscriptions/{id}", read.New(logger, subscriptionService).ServeHTTP)
			r.Delete("/subscriptions/{id}", remove.New(logger, subscriptionService).ServeHTTP)
			r.Put("/subscriptions/{id}", update.New(logger, subscriptionService).ServeHTTP)
			r.Get("/subscriptions/list", list.New(logger, subscriptionService).ServeHTTP)
			r.Post("/subscriptions/sum", sum.New(logger, subscriptionService).ServeHTTP)
			r.Post("/payment", paymentcreate.New(logger, providerClient, paymentService).ServeHTTP)
			r.Get("/payments/list", paymentlist.New(logger, paymentService).ServeHTTP)
		})

		// Webhook endpoint (без аутентификации)
		r.Post("/payments/webhook", paymentwebhook.New(logger, paymentService, senderService, "webhook_secret").ServeHTTP)
	})
	//r.Get("/health", health.New(logger).ServeHTTP)

	r.Handle("/metrics", promhttp.Handler())
	// Swagger docs endpoint
	r.Get("/docs/*", httpSwagger.WrapHandler)
}
