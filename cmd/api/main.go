package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go-baseline-skeleton/internal/baseline/app"
	"go-baseline-skeleton/internal/baseline/infra/config"
	"go-baseline-skeleton/internal/baseline/infra/idempotency"
	"go-baseline-skeleton/internal/baseline/infra/logging"
	"go-baseline-skeleton/internal/baseline/infra/tx"
	"go-baseline-skeleton/internal/baseline/transport/httpapi"
)

func main() {
	ctx := context.Background()

	cfgLoader := config.NewEnvLoader()
	cfg, err := cfgLoader.Load(ctx)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	logger := logging.NewJSONLogger(cfg.App.Name, cfg.App.Env)
	txManager := tx.NewNoopManager()
	idempotencyStore := idempotency.NewInMemoryStore()

	usecase := app.NewBootstrapUsecase(
		txManager,
		logger,
		cfg,
		nil, // repository
		nil, // cache
		nil, // mq
		nil, // websocket
		nil, // payment
		idempotencyStore,
	)

	if err := usecase.ValidateStartup(ctx); err != nil {
		log.Fatalf("startup validation failed: %v", err)
	}

	handler := httpapi.NewHandler(usecase, logger)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info(ctx, "server_start", map[string]any{"addr": cfg.HTTP.Addr})
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(ctx, "server_exit", err, map[string]any{"addr": cfg.HTTP.Addr})
	}
}