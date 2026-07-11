package bootstrap

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ambi/idmagic/backend/application"
	apppostgres "github.com/ambi/idmagic/backend/application/adapters/persistence/postgres"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres"
	oauth2valkey "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/valkey"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	valkeystore "github.com/ambi/idmagic/backend/shared/adapters/persistence/valkey"
	"github.com/ambi/idmagic/backend/shared/resilience"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedpostgres "github.com/ambi/idmagic/backend/wsfederation/adapters/persistence/postgres"
)

func assemblePostgresValkey(ctx context.Context) (*Dependencies, error) {
	databaseURL, valkeyURL := os.Getenv("DATABASE_URL"), os.Getenv("VALKEY_URL")
	if databaseURL == "" || valkeyURL == "" {
		return nil, errors.New("PERSISTENCE=postgres_valkey requires DATABASE_URL and VALKEY_URL")
	}

	// 1. レジリエンス構成のパラメータ構築
	dbCfg := postgres.DBConfig{
		MaxConns:        envInt32("DB_MAX_CONNS", 20),
		MinConns:        envInt32("DB_MIN_CONNS", 2),
		MaxConnIdleTime: envDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Second),
		MaxConnLifetime: envDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour),
		ConnectTimeout:  envDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
		QueryTimeout:    envDuration("DB_QUERY_TIMEOUT", 5*time.Second),
	}

	valkeyCfg := valkeystore.ValkeyConfig{
		DialTimeout:  envDuration("VALKEY_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  envDuration("VALKEY_READ_TIMEOUT", 2*time.Second),
		WriteTimeout: envDuration("VALKEY_WRITE_TIMEOUT", 2*time.Second),
		QueryTimeout: envDuration("VALKEY_QUERY_TIMEOUT", 2*time.Second),
	}

	// 2. サーキットブレイカーの構築
	dbBreaker := resilience.NewCircuitBreaker(resilience.Settings{ //nolint:contextcheck // Global breaker doesn't rely on request context
		Name:             "postgres",
		FailureThreshold: envFloat("DB_BREAKER_FAILURE_THRESHOLD", 0.5),
		Cooldown:         envDuration("DB_BREAKER_COOLDOWN", 30*time.Second),
		MinRequests:      envCircuitBreakerMinRequests("DB_BREAKER_MIN_REQUESTS"),
	})

	valkeyBreaker := resilience.NewCircuitBreaker(resilience.Settings{ //nolint:contextcheck // Global breaker doesn't rely on request context
		Name:             "valkey",
		FailureThreshold: envFloat("VALKEY_BREAKER_FAILURE_THRESHOLD", 0.5),
		Cooldown:         envDuration("VALKEY_BREAKER_COOLDOWN", 15*time.Second),
		MinRequests:      envCircuitBreakerMinRequests("VALKEY_BREAKER_MIN_REQUESTS"),
	})

	// 3. 接続オープン
	pool, err := postgres.Open(ctx, databaseURL, dbCfg)
	if err != nil {
		return nil, err
	}
	resilientDB := postgres.NewResilientDB(pool, dbBreaker, dbCfg.QueryTimeout)

	valkeyClient, err := valkeystore.Open(ctx, valkeyURL, valkeyCfg, valkeyBreaker)
	if err != nil {
		pool.Close()
		return nil, err
	}

	keyStore, err := postgres.NewKeyStore(ctx, resilientDB)
	if err != nil {
		pool.Close()
		_ = valkeyClient.Close()
		return nil, err
	}

	var sink oauthports.EventSink
	switch envDefault("EVENT_SINK", "console") {
	case "console":
		sink = eventsink.NewConsoleSink()
	case "outbox":
		sink = &oauth2postgres.OutboxEventSink{Pool: resilientDB}
	default:
		pool.Close()
		_ = valkeyClient.Close()
		return nil, errors.New("EVENT_SINK must be console or outbox")
	}

	return &Dependencies{
		ScimRepo:                &postgres.ScimRepository{Pool: resilientDB},
		TenantRepo:              &postgres.TenantRepository{Pool: resilientDB},
		AttrSchemaRepo:          &postgres.TenantUserAttributeSchemaRepository{Pool: resilientDB},
		UserRepo:                &postgres.UserRepository{Pool: resilientDB},
		GroupRepo:               &postgres.GroupRepository{Pool: resilientDB},
		AgentRepo:               &postgres.AgentRepository{Pool: resilientDB},
		MfaFactorRepo:           &postgres.MfaFactorRepository{Pool: resilientDB},
		PasswordHistoryRepo:     &postgres.PasswordHistoryRepository{Pool: resilientDB},
		PasswordResetTokenStore: &postgres.PasswordResetTokenStore{Pool: resilientDB},
		EmailChangeTokenStore:   &postgres.EmailChangeTokenStore{Pool: resilientDB},
		OAuth2: oauth2.Module{
			ClientRepo:                 &oauth2postgres.OAuth2ClientRepository{Pool: resilientDB},
			ConsentRepo:                &oauth2postgres.ConsentRepository{Pool: resilientDB},
			AuthzDetailTypeRepo:        &oauth2postgres.AuthorizationDetailTypeRepository{Pool: resilientDB},
			RequestStore:               &oauth2valkey.AuthorizationRequestStore{Client: valkeyClient},
			CodeStore:                  &oauth2valkey.AuthorizationCodeStore{Client: valkeyClient},
			PARStore:                   &oauth2valkey.PARStore{Client: valkeyClient},
			RefreshStore:               &oauth2postgres.RefreshTokenStore{Pool: resilientDB},
			DeviceCodeStore:            &oauth2valkey.DeviceCodeStore{Client: valkeyClient},
			DpopReplayStore:            &oauth2valkey.ReplayStore{Client: valkeyClient, Prefix: "dpop_replay:"},
			ClientAssertionReplayStore: &oauth2valkey.ReplayStore{Client: valkeyClient, Prefix: "client_assertion:"},
			AccessTokenDenylist:        &oauth2valkey.AccessTokenDenylist{Client: valkeyClient},
			AuditEventRepo:             &oauth2postgres.AuditEventRepository{Pool: resilientDB},
			EventSink:                  sink,
		},
		SessionStore:           &valkeystore.SessionStore{Client: valkeyClient},
		WebAuthnCredentialRepo: &postgres.WebAuthnCredentialRepository{Pool: resilientDB},
		WebAuthnSessionStore:   &valkeystore.WebAuthnSessionStore{Client: valkeyClient},
		RecoveryCodeRepo:       &postgres.RecoveryCodeRepository{Pool: resilientDB},
		NewLoginAttemptThrottle: func(configs authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle {
			return &valkeystore.LoginAttemptThrottle{Client: valkeyClient, Configs: configs}
		},
		KeyStore:             selectKeyStore(keyStore),
		TenantSaltStore:      postgres.NewTenantSaltStore(resilientDB),
		AuthEventBucketStore: &postgres.AuthEventBucketStore{Pool: resilientDB},
		WsFederation:         wsfederation.Module{RPRepo: &wsfedpostgres.WsFedRelyingPartyRepository{Pool: resilientDB}},
		SamlSPRepo:           &postgres.SamlServiceProviderRepository{Pool: resilientDB},
		Application: application.Module{
			Repo:                    &apppostgres.ApplicationRepository{Pool: resilientDB},
			IconStore:               &apppostgres.ApplicationIconStore{Pool: resilientDB},
			AssignmentRepo:          &apppostgres.ApplicationAssignmentRepository{Pool: resilientDB},
			OrderingRepo:            &apppostgres.ApplicationOrderingRepository{Pool: resilientDB},
			CategoryRepo:            &apppostgres.ApplicationCategoryRepository{Pool: resilientDB},
			SignInPolicyRepo:        &apppostgres.SignInPolicyRepository{Pool: resilientDB},
			DefaultSignInPolicyRepo: &apppostgres.DefaultSignInPolicyRepository{Pool: resilientDB},
		},
		Close: func() {
			_ = valkeyClient.Close()
			pool.Close()
		},
		DbPing: func(c context.Context) error {
			return pool.Ping(c)
		},
		ValkeyPing: func(c context.Context) error {
			return valkeyClient.Ping(c).Err()
		},
	}, nil
}
