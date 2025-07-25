package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/postgresql"
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
	_ = storage
	_ = ctx

	/*id, err := storage.CreateSubscriptionEntry(ctx, "YANDEX", 500, "60601fee-2bf1-4721-ae6f-7636e79a0cba", time.Date(2024, 1, 0, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		logger.Error("failed to create new entry", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
		os.Exit(1)
	}
	logger.Info("created new entry with ID", slog.String("id", strconv.Itoa(id)))
	rows, err := storage.RemoveSubscriptionEntryByUserID(ctx, "60601fee-2bf1-4721-ae6f-7636e79a0cba")
	if err != nil {
		logger.Error("failed to delete entrys by ID", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})
		os.Exit(1)
	}
	logger.Info("deleted entrys by ID", slog.String("rows", strconv.Itoa(int(rows))))*/
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

}
