package bootstrap

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

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	httpsupport "github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/adapters/observability"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/version"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Run はサーバ全体を起動する。SIGINT/SIGTERM で graceful shutdown。
func Run() error {
	runtime := loadRuntimeConfig()
	issuer := envDefault("ISSUER", "http://localhost:8080")
	addr := envDefault("ADDR", ":8080")

	shuttingDown := &atomic.Bool{}
	startupComplete := &atomic.Bool{}

	// アプリケーションログは stdout に構造化 JSON Lines で出力する (ADR-018)。
	// 監査ログ (DomainEvent) は EventSink 経由の別経路。
	buildInfo := version.Get()
	serviceName := envDefault("OTEL_SERVICE_NAME", "idmagic")
	logLevel := logging.ParseLevel(os.Getenv("LOG_LEVEL"))
	slogLogger := logging.NewSlog(os.Stdout, logLevel, serviceName, buildInfo.Version)
	logging.SetDefault(logging.New(os.Stdout, logLevel, serviceName, buildInfo.Version))
	logger := logging.Default()

	deps, err := assemble(context.Background())
	if err != nil {
		return fmt.Errorf("assemble dependencies: %w", err)
	}
	defer deps.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hasher := crypto.NewArgon2idPasswordHasher()
	if err := tenantusecases.EnsureDefault(ctx, deps.Tenancy.TenantRepo, time.Now().UTC()); err != nil {
		return fmt.Errorf("ensure default tenant: %w", err)
	}
	if os.Getenv("SKIP_DEMO_SEED") == "" {
		if err := seedDemoData(ctx, deps.OAuth2.ClientRepo, deps.IdentityManagement.UserRepo, deps.Authentication.MfaFactorRepo, deps.Authentication.PasswordHistoryRepo, deps.IdentityManagement.GroupRepo, deps.OAuth2.AuthzDetailTypeRepo, hasher); err != nil {
			return fmt.Errorf("seed demo data: %w", err)
		}
		if err := seedWsFedRelyingParty(ctx, deps.WsFederation.RPRepo); err != nil {
			return fmt.Errorf("seed federation relying party: %w", err)
		}
		if err := seedSamlServiceProvider(ctx, deps.Saml.SPRepo); err != nil {
			return fmt.Errorf("seed saml service provider: %w", err)
		}
		if err := seedDemoApplications(ctx, deps.Application.Repo, deps.Application.AssignmentRepo, time.Now().UTC()); err != nil {
			return fmt.Errorf("seed demo applications: %w", err)
		}
	}
	federationSigner, err := newDevFederationSigner()
	if err != nil {
		return fmt.Errorf("federation signer: %w", err)
	}
	sclDoc, err := spec.LoadSCL()
	if err != nil {
		return fmt.Errorf("load SCL: %w", err)
	}
	sentinelPasswordHash, err := hasher.Hash("idmagic-invalid-user-password")
	if err != nil {
		return fmt.Errorf("create sentinel password hash: %w", err)
	}
	emailSender, err := resolveEmailSender(os.Getenv)
	if err != nil {
		return fmt.Errorf("resolve email sender: %w", err)
	}
	breachedChecker, err := resolveBreachedPasswordChecker(os.Getenv)
	if err != nil {
		return fmt.Errorf("resolve breached password checker: %w", err)
	}
	objectiveInt := func(group, key string) int {
		value, ok := sclDoc.ObjectiveNestedInt("LoginThrottlePolicy", group, key)
		if !ok {
			return 0
		}
		return value
	}
	loginThrottle := deps.Authentication.NewLoginAttemptThrottle(authnports.LoginThrottleConfigs{
		Account: authnports.LoginThrottleConfig{
			MaxFailures:    objectiveInt("per_account", "max_failures"),
			WindowSeconds:  objectiveInt("per_account", "window_seconds"),
			LockoutSeconds: objectiveInt("per_account", "lockout_seconds"),
		},
		IP: authnports.LoginThrottleConfig{
			MaxFailures:    objectiveInt("per_ip", "max_failures"),
			WindowSeconds:  objectiveInt("per_ip", "window_seconds"),
			LockoutSeconds: objectiveInt("per_ip", "lockout_seconds"),
		},
	})
	authorizer, err := assembleAuthorizer()
	if err != nil {
		return err
	}
	sessionManager := authusecases.NewSessionManager(deps.Authentication.SessionStore)
	tokenSigner := crypto.NewJWTSigner(issuer, deps.KeyStore)
	jwkResolver := crypto.NewJWKResolver()
	deps.OAuth2.TokenIssuer = tokenSigner
	deps.OAuth2.TokenIntrospector = tokenSigner
	deps.OAuth2.Authorizer = authorizer
	deps.Authentication.PasswordHasher = hasher
	deps.Authentication.EmailSender = emailSender
	deps.Authentication.BreachedPasswordChecker = breachedChecker
	deps.Authentication.LoginAttemptThrottle = loginThrottle
	deps.Authentication.SentinelPasswordHash = sentinelPasswordHash
	deps.Authentication.SessionManager = sessionManager
	deps.Authentication.AuthnResolver = sessionManager

	e := echo.New()
	// Echo フレームワークのログも同じ構造化ハンドラ (ADR-018 の field 規約) に載せる。
	e.Logger = slogLogger
	// RequestFaultIsolation objective: request_id を最外で付与し、その内側で
	// panic を捕捉して 500 に局所化する。以降の otel / ハンドラの panic とログは
	// 同じ request_id 配下に入る。受信 X-Request-ID は secure-by-default で無視し
	// 自前生成する。信頼できる境界プロキシが所有・消毒している構成でのみ
	// REQUEST_ID_TRUST_INBOUND=true で受信値の再利用を許可する。
	e.Use(httpsupport.RequestIDMiddleware(envDefault("REQUEST_ID_TRUST_INBOUND", "false") == "true"))
	e.Use(httpsupport.RecoverMiddleware(logger))
	// SecurityResponseHeaders / FrameAncestorsPolicy objectives (ADR-076):
	// backend レスポンスへ CSP (nonce ベース) / frame-ancestors 'none' / nosniff 等を
	// 一元付与する。HSTS は TLS 終端層が所有するため既定は無効 (開発 http では抑制)。
	e.Use(httpsupport.SecurityHeadersMiddleware(loadSecurityHeaders(os.Getenv)))
	// HTTPServerHardening objective: ボディ上限を全リクエストに課し、超過は 413 で拒否する。
	// request_id 付与と panic recover の内側に置き、拒否レスポンスも相関/回復対象にする。
	hardening := loadHTTPServerHardening()
	e.Use(middleware.BodyLimit(hardening.MaxBodyBytes))
	var otelProvider *observability.Provider
	if runtime.Observability == "otel" {
		otelProvider, err = observability.New(ctx, envDefault("OTEL_SERVICE_NAME", "idmagic"), version.Get().Version)
		if err != nil {
			return fmt.Errorf("initialize OpenTelemetry: %w", err)
		}
		e.Use(otelProvider.Middleware)
	}
	emit := func(event spec.DomainEvent) {
		eventCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := deps.OAuth2.EventSink.Emit(eventCtx, event); err != nil {
			logger.Error(eventCtx, "event sink emit failed", "error", err)
		}
		if deps.Audit.AuditEventRepo != nil {
			if rec, err := newAuditEventRecord(event); err == nil {
				appendCtx := eventCtx
				if rec.TenantID != "" {
					appendCtx = tenancy.WithTenant(eventCtx, &tenancydomain.Tenant{ID: rec.TenantID}, "", "")
				}
				_ = deps.Audit.AuditEventRepo.Append(appendCtx, rec)
			}
		}
	}
	httpadapter.Register(e, httpadapter.Deps{
		Deps: httpsupport.Deps{
			Issuer:                    issuer,
			SCL:                       sclDoc,
			LegacyBareIssuer:          envDefault("LEGACY_BARE_ISSUER", "false") == "true",
			TrustedForwardedHops:      envInt("TRUSTED_FORWARDED_HOPS", 0),
			OperationTimeout:          0, // 必要なら設定
			DetachedCompletionTimeout: 0,
			Emit:                      emit,
			DbPing:                    deps.DbPing,
			ValkeyPing:                deps.ValkeyPing,
			ShuttingDown:              shuttingDown,
			StartupComplete:           startupComplete,
			TenantRepo:                deps.Tenancy.TenantRepo,
			HealthInfo: httpsupport.HealthInfo{
				Persistence:   runtime.Persistence,
				EventSink:     runtime.EventSink,
				Observability: runtime.Observability,
				AuthZEN:       runtime.AuthZEN,
			},
		},
		Tenancy:            deps.Tenancy,
		IdentityManagement: deps.IdentityManagement,
		Authentication:     deps.Authentication,
		OAuth2:             deps.OAuth2,
		KeyStore:           deps.KeyStore,
		Audit:              deps.Audit,
		JWKResolver:        jwkResolver,
		WsFederation:       deps.WsFederation,
		Saml:               deps.Saml,
		Scim:               deps.Scim,
		FederationSigner:   federationSigner,
		Application:        deps.Application,
	})

	startRetentionSweep(ctx, deps, envDuration("RETENTION_SWEEP_INTERVAL", time.Hour))

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
			hardening.apply(s)
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
		drainGracePeriod := 5 * time.Second
		if val := os.Getenv("DRAIN_GRACE_PERIOD_SECONDS"); val != "" {
			if parsed, err := time.ParseDuration(val + "s"); err == nil {
				drainGracePeriod = parsed
			}
		}
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
	return runErr
}
