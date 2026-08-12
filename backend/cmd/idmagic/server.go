package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"

	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	cimdhttp "github.com/ambi/idmagic/backend/oauth2/client/cimd_http"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	httpsupport "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	metricsPrometheus "github.com/ambi/idmagic/backend/shared/observability/metrics_prometheus"
	telemetryOTLP "github.com/ambi/idmagic/backend/shared/observability/telemetry_otlp"
	passwordsArgon2id "github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/version"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Run はサーバ全体を起動する。SIGINT/SIGTERM で graceful shutdown。
func Run() error {
	// 起動時設定は Assemble や listener 起動より前に一括で読み込み検証する。
	// 必須値欠落・型/範囲不正・相互矛盾する組み合わせはここで集約エラーとして
	// 返し、部分起動させない (REQ-SYSTEM-016)。
	loader := bootstrap.NewConfigLoader(os.Getenv)
	shared := bootstrap.LoadSharedConfig(loader)
	api := bootstrap.LoadAPIConfig(loader)
	seed := bootstrap.LoadSeedConfig(loader)
	if err := loader.Err(); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}

	runtime := bootstrap.LoadRuntimeConfig(shared)
	issuer := api.Issuer
	addr := api.Addr

	shuttingDown := &atomic.Bool{}
	startupComplete := &atomic.Bool{}

	// アプリケーションログは stdout に構造化 JSON Lines で出力する。
	// 監査ログ (DomainEvent) は EventSink 経由の別経路。
	buildInfo := version.Get()
	serviceName := api.OTelServiceName
	logLevel := logging.ParseLevel(api.LogLevel)
	slogLogger := logging.NewSlog(os.Stdout, logLevel, serviceName, buildInfo.Version)
	logging.SetDefault(logging.New(os.Stdout, logLevel, serviceName, buildInfo.Version))
	logger := logging.Default()

	deps, err := bootstrap.Assemble(context.Background(), shared)
	if err != nil {
		return fmt.Errorf("assemble dependencies: %w", err)
	}
	defer deps.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hasher := passwordsArgon2id.NewArgon2idPasswordHasher()
	if err := tenantusecases.EnsureDefault(ctx, deps.Tenancy.TenantRepo, time.Now().UTC()); err != nil {
		return fmt.Errorf("ensure default tenant: %w", err)
	}
	if seed.Configured() {
		if _, err := bootstrap.Seed(ctx, deps, seed.Request(), seed.SecretRoot); err != nil {
			return fmt.Errorf("explicit startup seed: %w", err)
		}
	}
	federationSigner := samltoken.KeyStoreSignerProvider{KeyStore: deps.SigningKeys.KeyStore}
	runtimeContract := spec.CurrentRuntimeContract()
	sentinelPasswordHash, err := hasher.Hash("idmagic-invalid-user-password")
	if err != nil {
		return fmt.Errorf("create sentinel password hash: %w", err)
	}
	breachedChecker := bootstrap.ResolveBreachedPasswordChecker(shared)
	loginThrottle := deps.Authentication.NewLoginAttemptThrottle(sessionports.LoginThrottleConfigs{
		Account: sessionports.LoginThrottleConfig{
			MaxFailures:    10,
			WindowSeconds:  900,
			LockoutSeconds: 900,
		},
		IP: sessionports.LoginThrottleConfig{
			MaxFailures:    30,
			WindowSeconds:  900,
			LockoutSeconds: 900,
		},
	})
	rateLimiter := deps.RateLimit.NewRateLimiter(api.RateLimits)
	authorizer := bootstrap.AssembleAuthorizer(shared)
	sessionManager := sessionusecases.NewSessionManager(deps.Authentication.SessionStore)
	tokenSigner := tokensJOSE.NewJWTSigner(issuer, deps.SigningKeys.KeyStore)
	jwkResolver := tokensJOSE.NewJWKResolver()
	managedTokenIntrospector := apitokenusecases.New(
		deps.ApiTokens.Repo,
		apitokenusecases.WithTokenIntrospector(tokenSigner),
	)
	deps.ApiTokens.TokenIssuer = tokenSigner
	deps.ApiTokens.TokenIntrospector = tokenSigner
	deps.OAuth2.TokenIssuer = tokenSigner
	deps.OAuth2.TokenIntrospector = managedTokenIntrospector
	deps.OAuth2.IDTokenHintVerifier = tokenSigner
	deps.OAuth2.Authorizer = authorizer
	deps.Authentication.PasswordHasher = hasher
	deps.Authentication.BreachedPasswordChecker = breachedChecker
	deps.Authentication.LoginAttemptThrottle = loginThrottle
	deps.RateLimit.RateLimiter = rateLimiter
	deps.Authentication.SentinelPasswordHash = sentinelPasswordHash
	deps.Authentication.SessionManager = sessionManager
	deps.Authentication.AuthnResolver = sessionManager

	e := echo.New()

	// MetricsExposition objective: pull-based /metrics は OTLP collector の有無に
	// 依存しないため、OBSERVABILITY 設定 (OTLP push tracing/metrics) とは独立に
	// 常時構築する。RED middleware はここで組み立てた Meter へ記録する。
	appMetrics, err := metricsPrometheus.NewMetrics(api.OTelServiceName, version.Get().Version)
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}

	// Echo フレームワークのログも同じ構造化ハンドラ (field 規約) に載せる。
	e.Logger = slogLogger
	// DefaultHTTPErrorHandler は仕様上エラーをログに残さない。ハンドラが返す
	// 生エラー (panic ではないもの) が 500 になったとき原因を追えるようにする。
	e.HTTPErrorHandler = httpsupport.ErrorHandler(logger, appMetrics)
	// RequestFaultIsolation objective: request_id を最外で付与し、その内側で
	// panic を捕捉して 500 に局所化する。以降の otel / ハンドラの panic とログは
	// 同じ request_id 配下に入る。受信 X-Request-ID は secure-by-default で無視し
	// 自前生成する。信頼できる境界プロキシが所有・消毒している構成でのみ
	// REQUEST_ID_TRUST_INBOUND=true で受信値の再利用を許可する。
	e.Use(httpsupport.RequestIDMiddleware(api.RequestIDTrustInbound))
	e.Use(httpsupport.LoggingMiddleware())
	e.Use(httpsupport.RecoverMiddleware(logger))
	// SecurityResponseHeaders / FrameAncestorsPolicy objectives:
	// backend レスポンスへ CSP (nonce ベース) / frame-ancestors 'none' / nosniff 等を
	// 一元付与する。HSTS は TLS 終端層が所有するため既定は無効 (開発 http では抑制)。
	e.Use(httpsupport.SecurityHeadersMiddleware(api.SecurityHeaders))
	// HTTPServerHardening objective: ボディ上限を全リクエストに課し、超過は 413 で拒否する。
	// request_id 付与と panic recover の内側に置き、拒否レスポンスも相関/回復対象にする。
	e.Use(httpsupport.MetricsMiddleware(appMetrics))
	hardening := api.Hardening
	e.Use(middleware.BodyLimit(hardening.MaxBodyBytes))
	var otelProvider *telemetryOTLP.Provider
	if runtime.Observability == "otel" {
		otelProvider, err = telemetryOTLP.New(ctx, api.OTelServiceName, version.Get().Version)
		if err != nil {
			return fmt.Errorf("initialize OpenTelemetry: %w", err)
		}
		e.Use(otelProvider.Middleware)
	}
	emit := deps.NewEmitFunc(logger)
	sessionManager.QuotaRepo = deps.Tenancy.QuotaRepo
	sessionManager.Emit = emit
	// ClientRepo is wrapped with CIMD resolution in bootstrap; Emit can only
	// be wired here, after NewEmitFunc exists (same reason as sessionManager.Emit above).
	if cimdRepo, ok := deps.OAuth2.ClientRepo.(*cimdhttp.ClientRepositoryWithCIMD); ok {
		cimdRepo.Emit = emit
	}
	httpadapter.Register(e, httpadapter.Deps{
		MetricsHandler: appMetrics.Handler(),
		Deps: httpsupport.Deps{
			Issuer:                    issuer,
			Contract:                  runtimeContract,
			TenantBaseDomain:          api.TenantBaseDomain,
			TrustedForwardedHops:      api.TrustedForwardedHops,
			RateLimiter:               deps.RateLimit.RateLimiter,
			OperationTimeout:          0, // 必要なら設定
			DetachedCompletionTimeout: 0,
			AbortMetrics:              appMetrics,
			Metrics:                   appMetrics,
			Emit:                      emit,
			DbPing:                    deps.DbPing,
			ShuttingDown:              shuttingDown,
			StartupComplete:           startupComplete,
			TenantRepo:                deps.Tenancy.TenantRepo,
			PaginationCodec:           httpsupport.NewCursorCodec(bootstrap.PaginationCursorSecret(api.PaginationCursorSecret)),
			HealthInfo: httpsupport.HealthInfo{
				Persistence:   runtime.Persistence,
				Observability: runtime.Observability,
				AuthZEN:       runtime.AuthZEN,
			},
		},
		Tenancy:          deps.Tenancy,
		IdManagement:     deps.IdManagement,
		IdGovernance:     deps.IdGovernance,
		Authentication:   deps.Authentication,
		Notification:     deps.Notification,
		OAuth2:           deps.OAuth2,
		SigningKeys:      deps.SigningKeys,
		DataKeys:         deps.DataKeys,
		Audit:            deps.Audit,
		JWKResolver:      jwkResolver,
		WsFederation:     deps.WsFederation,
		Saml:             deps.Saml,
		Sourcing:         deps.Sourcing,
		FederationSigner: federationSigner,
		Application:      deps.Application,
		ApiTokens:        deps.ApiTokens,
		Jobs:             deps.Jobs,
		Provisioning:     deps.Provisioning,
		WorkloadIdentity: deps.WorkloadIdentity,
		SharedSignals:    deps.SharedSignals,
	})

	// 起動準備がすべて完了したので、startupComplete を true に設定する
	startupComplete.Store(true)

	logger.Info(ctx, "server listening",
		"commit", buildInfo.GitCommit, "build_date", buildInfo.BuildDate,
		"addr", addr, "issuer", issuer,
		"read_header_timeout", hardening.ReadHeaderTimeout, "read_timeout", hardening.ReadTimeout,
		"write_timeout", hardening.WriteTimeout, "idle_timeout", hardening.IdleTimeout,
		"max_body_bytes", hardening.MaxBodyBytes)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	serverErrChan := make(chan error, 1)
	startConfig := echo.StartConfig{
		Address: addr,
		// HTTPServerHardening objective: 基盤 http.Server にタイムアウトを設定する。
		BeforeServeFunc: func(s *http.Server) error {
			hardening.Apply(s)
			return nil
		},
	}
	go func() {
		if err := startConfig.Start(serverCtx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- err
		} else {
			serverErrChan <- nil
		}
	}()

	// シグナルを明示的に待ち受ける
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var runErr error
	select {
	case sig := <-sigChan:
		logger.Info(context.Background(), "received signal, starting graceful drain", "signal", sig.String())
		// 1. readiness probe を unready に落とす
		shuttingDown.Store(true)

		// 2. ドレイン猶予待機
		drainGracePeriod := time.Duration(api.DrainGracePeriod) * time.Second
		logger.Info(context.Background(), "waiting for connection drain", "duration", drainGracePeriod.String())
		time.Sleep(drainGracePeriod)

		// 3. サーバシャットダウン
		logger.Info(context.Background(), "stopping server")
		serverCancel()
		runErr = <-serverErrChan

	case err := <-serverErrChan:
		runErr = err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if otelProvider != nil {
		if err := otelProvider.Shutdown(shutdownCtx); err != nil {
			logger.Error(shutdownCtx, "shutdown OpenTelemetry failed", "error", err)
		}
	}
	if err := appMetrics.Shutdown(shutdownCtx); err != nil {
		logger.Error(shutdownCtx, "shutdown metrics failed", "error", err)
	}
	return runErr
}
