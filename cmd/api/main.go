package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	baselineapp "go-baseline-skeleton/internal/baseline/app"
	baselinedomain "go-baseline-skeleton/internal/baseline/domain"
	baselineconfig "go-baseline-skeleton/internal/baseline/infra/config"
	baselineidempotency "go-baseline-skeleton/internal/baseline/infra/idempotency"
	baselinelogging "go-baseline-skeleton/internal/baseline/infra/logging"
	baselinetx "go-baseline-skeleton/internal/baseline/infra/tx"
	baselinehttpapi "go-baseline-skeleton/internal/baseline/transport/httpapi"

	cartapp "go-baseline-skeleton/internal/cart/app"
	cartgateway "go-baseline-skeleton/internal/cart/infra/gateway"
	cartidempotency "go-baseline-skeleton/internal/cart/infra/idempotency"
	cartrepo "go-baseline-skeleton/internal/cart/infra/repo"
	carttx "go-baseline-skeleton/internal/cart/infra/tx"
	carthttpapi "go-baseline-skeleton/internal/cart/transport/httpapi"

	identityapp "go-baseline-skeleton/internal/identity/app"
	identitydomain "go-baseline-skeleton/internal/identity/domain"
	identitycontext "go-baseline-skeleton/internal/identity/infra/contextx"
	identityjwt "go-baseline-skeleton/internal/identity/infra/jwt"
	identitymiddleware "go-baseline-skeleton/internal/identity/infra/middleware"
	identitypassword "go-baseline-skeleton/internal/identity/infra/password"
	identityrepo "go-baseline-skeleton/internal/identity/infra/repo"
	identitysession "go-baseline-skeleton/internal/identity/infra/session"
	identitytx "go-baseline-skeleton/internal/identity/infra/tx"
	identityhttpapi "go-baseline-skeleton/internal/identity/transport/httpapi"

	productapp "go-baseline-skeleton/internal/product/app"
	productcache "go-baseline-skeleton/internal/product/infra/cache"
	productidempotency "go-baseline-skeleton/internal/product/infra/idempotency"
	productrepo "go-baseline-skeleton/internal/product/infra/repo"
	producttx "go-baseline-skeleton/internal/product/infra/tx"
	producthttpapi "go-baseline-skeleton/internal/product/transport/httpapi"

	reportapp "go-baseline-skeleton/internal/report/app"
	reportcache "go-baseline-skeleton/internal/report/infra/cache"
	reportddl "go-baseline-skeleton/internal/report/infra/ddl"
	reportrepo "go-baseline-skeleton/internal/report/infra/repo"
	reportrouter "go-baseline-skeleton/internal/report/infra/router"
	reporttx "go-baseline-skeleton/internal/report/infra/tx"
	reporthttpapi "go-baseline-skeleton/internal/report/transport/httpapi"

	paymentcallbackapp "go-baseline-skeleton/internal/payment_callback/app"
	paymentcallbackidempotency "go-baseline-skeleton/internal/payment_callback/infra/idempotency"
	paymentcallbackmq "go-baseline-skeleton/internal/payment_callback/infra/mq"
	paymentcallbackrepo "go-baseline-skeleton/internal/payment_callback/infra/repo"
	paymentcallbacktx "go-baseline-skeleton/internal/payment_callback/infra/tx"
	paymentcallbackverify "go-baseline-skeleton/internal/payment_callback/infra/verify"
	paymentcallbackhttpapi "go-baseline-skeleton/internal/payment_callback/transport/httpapi"

	orderapp "go-baseline-skeleton/internal/order/app"
	ordercache "go-baseline-skeleton/internal/order/infra/cache"
	ordercart "go-baseline-skeleton/internal/order/infra/cart"
	orderidempotency "go-baseline-skeleton/internal/order/infra/idempotency"
	ordermq "go-baseline-skeleton/internal/order/infra/mq"
	orderpayment "go-baseline-skeleton/internal/order/infra/payment"
	orderrepo "go-baseline-skeleton/internal/order/infra/repo"
	orderrouter "go-baseline-skeleton/internal/order/infra/router"
	ordertx "go-baseline-skeleton/internal/order/infra/tx"
	orderws "go-baseline-skeleton/internal/order/infra/websocket"
	orderhttpapi "go-baseline-skeleton/internal/order/transport/httpapi"
)

func main() {
	ctx := context.Background()

	cfgLoader := baselineconfig.NewEnvLoader()
	cfg, err := cfgLoader.Load(ctx)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	logger := baselinelogging.NewJSONLogger(cfg.App.Name, cfg.App.Env)

	var baselineTxManager baselinedomain.TxManager = baselinetx.NewNoopManager()
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
		baselineTxManager = baselinetx.NewSQLManager(db, nil)
	}

	redisClient := baselineidempotency.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if cfg.Idempotency.Enabled {
		if err = redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("ping redis failed: %v", err)
		}
	}
	defer redisClient.Close()

	idempotencyStore := baselineidempotency.NewRedisStore(redisClient, cfg.Redis.KeyPrefix)
	baselineUsecase := baselineapp.NewBootstrapUsecase(
		baselineTxManager,
		logger,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		idempotencyStore,
	)
	if err := baselineUsecase.ValidateStartup(ctx); err != nil {
		log.Fatalf("startup validation failed: %v", err)
	}

	baselineHandler := baselinehttpapi.NewHandler(baselineUsecase, logger)
	identityHandler := buildIdentityHandler(db, redisClient)
	productHandler := buildProductHandler(db, redisClient)
	cartHandler := buildCartHandler(db, redisClient)
	reportHandler := buildReportHandler(db, redisClient)
	paymentCallbackHandler := buildPaymentCallbackHandler(db, redisClient)
	orderHandler := buildOrderHandler(db, redisClient)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           composeRoutes(identityHandler.Routes(), productHandler.Routes(), cartHandler.Routes(), reportHandler.Routes(), paymentCallbackHandler.Routes(), orderHandler.Routes(), baselineHandler.Routes()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info(ctx, "server_start", map[string]any{"addr": cfg.HTTP.Addr})
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(ctx, "server_exit", err, map[string]any{"addr": cfg.HTTP.Addr})
	}
}

func buildIdentityHandler(db *sql.DB, redisClient redis.UniversalClient) *identityhttpapi.Handler {
	principalCtx := identitycontext.NewPrincipalStore()
	passwordSvc := identitypassword.NewMD5Comparator()

	var repo identitydomain.AccountRepository
	if db != nil {
		repo = identityrepo.NewSQLAccountRepo(db)
	} else {
		repo = identityrepo.NewInMemoryAccountRepo([]*identitydomain.Account{
			{ID: 1, Type: identitydomain.AccountTypeEmployee, Username: "admin", DisplayName: "Admin", PasswordHash: identitypassword.HashMD5("123456"), Status: identitydomain.AccountStatusEnabled},
		})
	}

	tokenSvc := identityjwt.NewTokenService(identityjwt.Config{
		Algorithm: "HS256",
		Employee: identityjwt.AccountJWTConfig{Secret: readOrDefault("IDENTITY_JWT_ADMIN_SECRET", "itcast"), Issuer: strings.TrimSpace(os.Getenv("IDENTITY_JWT_ADMIN_ISSUER")), TTL: time.Duration(envInt64("IDENTITY_JWT_ADMIN_TTL_MS", 720000000)) * time.Millisecond, ClaimKey: "empId"},
		User:     identityjwt.AccountJWTConfig{Secret: readOrDefault("IDENTITY_JWT_USER_SECRET", "itcast"), Issuer: strings.TrimSpace(os.Getenv("IDENTITY_JWT_USER_ISSUER")), TTL: time.Duration(envInt64("IDENTITY_JWT_USER_TTL_MS", 720000000)) * time.Millisecond, ClaimKey: "userId"},
	})

	authSvc := identityapp.NewAuthService(identityapp.AuthDeps{
		Repo:              repo,
		Token:             tokenSvc,
		Password:          passwordSvc,
		PrincipalCtx:      principalCtx,
		Tx:                identitytx.NewNoopManager(),
		Sessions:          identitysession.NewRedisStore(redisClient, readOrDefault("IDENTITY_SESSION_REDIS_PREFIX", "identity:session")),
		RevokeAllOnLogout: envBool("IDENTITY_REVOKE_ALL_ON_LOGOUT", true),
	})

	authMiddleware := identitymiddleware.NewRequireAuth(authSvc, principalCtx)
	return identityhttpapi.NewHandler(authSvc, principalCtx, authMiddleware)
}

func buildProductHandler(db *sql.DB, redisClient redis.UniversalClient) *producthttpapi.Handler {
	readRepo := productrepo.NewMySQLReadRepository(db)
	readCache := productcache.NewRedisReadCache(redisClient, strings.TrimSpace(os.Getenv("PRODUCT_CACHE_NAMESPACE")))
	readSvc := productapp.NewReadService(productapp.ReadDeps{
		Repo:        readRepo,
		Cache:       readCache,
		CategoryTTL: time.Duration(envInt64("PRODUCT_CACHE_CATEGORY_TTL_SEC", 300)) * time.Second,
		DishTTL:     time.Duration(envInt64("PRODUCT_CACHE_DISH_TTL_SEC", 300)) * time.Second,
		SetmealTTL:  time.Duration(envInt64("PRODUCT_CACHE_SETMEAL_TTL_SEC", 300)) * time.Second,
	})

	writeRepo := productrepo.NewMySQLWriteRepository(db)
	invalidator := productcache.NewRedisInvalidator(readCache)
	outbox := productcache.NewRedisInvalidationOutbox(redisClient, readOrDefault("PRODUCT_CACHE_INVALIDATION_OUTBOX_KEY", "product:cache_invalidation:outbox"))

	var writeIdemStore interface {
		Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
		MarkDone(ctx context.Context, scene, key, token string, result []byte) error
		MarkFailed(ctx context.Context, scene, key, token, reason string) error
		GetDoneResult(ctx context.Context, scene, key string) (result []byte, found bool, err error)
	}
	if redisClient != nil {
		writeIdemStore = productidempotency.NewRedisStore(redisClient, readOrDefault("PRODUCT_IDEMPOTENCY_REDIS_PREFIX", "product:idempotency"))
	} else {
		writeIdemStore = productidempotency.NewInMemoryStore()
	}

	var txManager interface {
		RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
	txManager = producttx.NewNoopManager()
	if db != nil {
		txManager = producttx.NewSQLManager(db, nil)
	}

	writeSvc := productapp.NewWriteService(productapp.WriteDeps{
		Repo:           writeRepo,
		Tx:             txManager,
		Invalidator:    invalidator,
		Idempotency:    writeIdemStore,
		Outbox:         outbox,
		IdempotencyTTL: time.Duration(envInt64("PRODUCT_WRITE_IDEMPOTENCY_TTL_SEC", 300)) * time.Second,
	})
	adminHandler := producthttpapi.NewAdminHandler(writeSvc)
	return producthttpapi.NewHandler(readSvc, adminHandler)
}

func buildCartHandler(db *sql.DB, redisClient redis.UniversalClient) *carthttpapi.Handler {
	cartReadRepo := productrepo.NewMySQLReadRepository(db)
	products := cartgateway.NewProductGateway(cartReadRepo)
	users := cartgateway.NewUserContext()
	repo := cartrepo.NewMySQLCartRepo(db)

	var txManager interface {
		RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
	txManager = carttx.NewNoopManager()
	if db != nil {
		txManager = carttx.NewSQLManager(db, nil)
	}

	var idemStore interface {
		Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
		MarkDone(ctx context.Context, scene, key, token string, result []byte) error
		MarkFailed(ctx context.Context, scene, key, token, reason string) error
		GetDoneResult(ctx context.Context, scene, key string) (result []byte, found bool, err error)
	}
	if redisClient != nil {
		idemStore = cartidempotency.NewRedisStore(redisClient, readOrDefault("CART_IDEMPOTENCY_REDIS_PREFIX", "cart:idempotency"))
	} else if db != nil {
		idemStore = cartidempotency.NewSQLStore(db)
	} else {
		idemStore = cartidempotency.NewInMemoryStore()
	}

	svc := cartapp.NewService(cartapp.Deps{
		Repo:           repo,
		Products:       products,
		Users:          users,
		Tx:             txManager,
		Idempotency:    idemStore,
		IdempotencyTTL: time.Duration(envInt64("CART_IDEMPOTENCY_TTL_SEC", 300)) * time.Second,
	})
	return carthttpapi.NewHandler(svc)
}

func buildReportHandler(db *sql.DB, redisClient redis.UniversalClient) *reporthttpapi.Handler {
	repo := reportrepo.NewMySQLReportRepo(db)
	cache := reportcache.NewRedisReportCache(redisClient, readOrDefault("REPORT_CACHE_NAMESPACE", "report:cache"))

	shardingEnabled := envBool("REPORT_SHARDING_ENABLED", true)
	scanMonths := int(envInt64("REPORT_SHARDING_SCAN_MONTHS", 0))
	router := reportrouter.NewMonthShardRouterWithOptions(
		readOrDefault("REPORT_ORDER_BASE_TABLE", "orders"),
		shardingEnabled,
		scanMonths,
	)

	svc := reportapp.NewService(reportapp.Deps{
		Repo:        repo,
		Router:      router,
		Cache:       cache,
		Tx:          reporttx.NewNoopManager(),
		DDL:         reportddl.NewShardTableManager(db, readOrDefault("REPORT_ORDER_BASE_TABLE", "orders")),
		OverviewTTL: time.Duration(envInt64("REPORT_CACHE_OVERVIEW_TTL_SEC", 120)) * time.Second,
		TrendTTL:    time.Duration(envInt64("REPORT_CACHE_TREND_TTL_SEC", 120)) * time.Second,
		ListTTL:     time.Duration(envInt64("REPORT_CACHE_LIST_TTL_SEC", 30)) * time.Second,
	})
	return reporthttpapi.NewHandler(svc)
}

func buildPaymentCallbackHandler(db *sql.DB, redisClient redis.UniversalClient) *paymentcallbackhttpapi.Handler {
	var txManager interface {
		RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
	txManager = paymentcallbacktx.NewNoopManager()
	if db != nil {
		txManager = paymentcallbacktx.NewSQLManager(db, nil)
	}

	var idemStore interface {
		Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
		MarkDone(ctx context.Context, scene, key, token string) error
		MarkFailed(ctx context.Context, scene, key, token, reason string) error
	}
	if redisClient != nil {
		idemStore = paymentcallbackidempotency.NewRedisStore(redisClient, readOrDefault("PAYMENT_CALLBACK_IDEMPOTENCY_PREFIX", "payment_callback:idempotency"))
	} else {
		idemStore = paymentcallbackidempotency.NewInMemoryStore()
	}

	usecase := paymentcallbackapp.NewService(paymentcallbackapp.Deps{
		Verifier:       paymentcallbackverify.NewChannelVerifier(parseChannelSecretsEnv(os.Getenv("PAYMENT_CALLBACK_CHANNEL_SECRETS"))),
		Repo:           paymentcallbackrepo.NewMySQLCallbackRepo(db),
		Idempotency:    idemStore,
		Publisher:      paymentcallbackmq.NewEventPublisher(db),
		GrayPolicy:     paymentcallbackverify.NewSimpleGrayPolicy(envBool("PAYMENT_CALLBACK_GRAY_ENABLED", true)),
		Tx:             txManager,
		IdempotencyTTL: time.Duration(envInt64("PAYMENT_CALLBACK_IDEMPOTENCY_TTL_SEC", 86400)) * time.Second,
	})
	return paymentcallbackhttpapi.NewHandler(usecase)
}

func buildOrderHandler(db *sql.DB, redisClient redis.UniversalClient) *orderhttpapi.Handler {
	writeRouter := orderrouter.NewMonthShardRouter(readOrDefault("ORDER_BASE_TABLE", "orders"), envBool("ORDER_SHARDING_ENABLED", true))
	repo := orderrepo.NewMySQLOrderRepo(db, writeRouter)
	cartReader := ordercart.NewReader(
		db,
		envInt64("ORDER_DELIVERY_FEE", 0),
		envInt64("ORDER_COUPON_DISCOUNT", 0),
		envInt64("ORDER_FULL_REDUCTION_TRIGGER", 0),
		envInt64("ORDER_FULL_REDUCTION_AMOUNT", 0),
	)
	paymentGateway := orderpayment.NewGateway()
	cacheInvalidator := ordercache.NewNoopInvalidator()
	mqPublisher := ordermq.NewPublisher(db)
	wsNotifier := orderws.NewNotifier()

	var txManager interface {
		RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
	txManager = ordertx.NewNoopManager()
	if db != nil {
		txManager = ordertx.NewSQLManager(db, nil)
	}

	var idemStore interface {
		Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
		MarkDone(ctx context.Context, scene, key, token string, result []byte) error
		MarkFailed(ctx context.Context, scene, key, token, reason string) error
		GetDoneResult(ctx context.Context, scene, key string) (result []byte, found bool, err error)
	}
	if redisClient != nil {
		idemStore = orderidempotency.NewRedisStore(redisClient, readOrDefault("ORDER_IDEMPOTENCY_REDIS_PREFIX", "order:idempotency"))
	} else {
		idemStore = orderidempotency.NewInMemoryStore()
	}

	svc := orderapp.NewService(orderapp.Deps{
		Repo:           repo,
		Cart:           cartReader,
		Payment:        paymentGateway,
		Cache:          cacheInvalidator,
		MQ:             mqPublisher,
		WebSocket:      wsNotifier,
		Idempotency:    idemStore,
		Tx:             txManager,
		IdempotencyTTL: time.Duration(envInt64("ORDER_IDEMPOTENCY_TTL_SEC", 86400)) * time.Second,
	})
	return orderhttpapi.NewHandler(svc)
}

func composeRoutes(identityRoutes, productRoutes, cartRoutes, reportRoutes, paymentCallbackRoutes, orderRoutes, baselineRoutes http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/identity/") || r.URL.Path == "/identity" {
			identityRoutes.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/report/") || r.URL.Path == "/report" ||
			strings.HasPrefix(r.URL.Path, "/admin/report/") || r.URL.Path == "/admin/report" {
			reportRoutes.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/product/") || r.URL.Path == "/product" || strings.HasPrefix(r.URL.Path, "/admin/") || r.URL.Path == "/admin" {
			productRoutes.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cart/") || r.URL.Path == "/cart" {
			cartRoutes.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/payment/") || r.URL.Path == "/payment" {
			paymentCallbackRoutes.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/orders") || r.URL.Path == "/orders" {
			orderRoutes.ServeHTTP(w, r)
			return
		}
		baselineRoutes.ServeHTTP(w, r)
	})
}

func parseChannelSecretsEnv(raw string) map[string]string {
	out := make(map[string]string)
	for _, seg := range strings.Split(raw, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		parts := strings.SplitN(seg, ":", 2)
		if len(parts) != 2 {
			continue
		}
		channel := strings.ToUpper(strings.TrimSpace(parts[0]))
		secret := strings.TrimSpace(parts[1])
		if channel == "" || secret == "" {
			continue
		}
		out[channel] = secret
	}
	return out
}

func readOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt64(key string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return parsed
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}




