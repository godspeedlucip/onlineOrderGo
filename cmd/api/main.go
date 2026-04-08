package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"go-baseline-skeleton/internal/baseline/app"
	"go-baseline-skeleton/internal/baseline/domain"
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

	var txManager domain.TxManager = tx.NewNoopManager()
	var db *sql.DB
	if cfg.DB.DSN != "" {
		db, err = sql.Open(cfg.DB.Driver, cfg.DB.DSN)
		if err != nil {
			log.Fatalf("open db failed: %v", err)
		}
		if err = db.PingContext(ctx); err != nil {
			log.Fatalf("ping db failed: %v", err)
		}
		defer db.Close()
		txManager = tx.NewSQLManager(db, nil)
	}

	redisClient := idempotency.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if cfg.Idempotency.Enabled {
		if err = redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("ping redis failed: %v", err)
		}
	}
	defer redisClient.Close()
	idempotencyStore := idempotency.NewRedisStore(redisClient, cfg.Redis.KeyPrefix)

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
