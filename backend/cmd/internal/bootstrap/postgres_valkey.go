package bootstrap

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ambi/idmagic/backend/application"
	apppostgres "github.com/ambi/idmagic/backend/application/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/audit"
	auditpostgres "github.com/ambi/idmagic/backend/audit/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/authentication"
	authnpostgres "github.com/ambi/idmagic/backend/authentication/adapters/persistence/postgres"
	authnvalkey "github.com/ambi/idmagic/backend/authentication/adapters/persistence/valkey"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/identitymanagement"
	idmpostgres "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/jobs"
	jobspostgres "github.com/ambi/idmagic/backend/jobs/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres"
	oauth2valkey "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/valkey"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/saml"
	samlpostgres "github.com/ambi/idmagic/backend/saml/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/scim"
	scimpostgres "github.com/ambi/idmagic/backend/scim/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	sharedvalkey "github.com/ambi/idmagic/backend/shared/adapters/persistence/valkey"
	"github.com/ambi/idmagic/backend/shared/resilience"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
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
		MaxConnIdleTime: EnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Second),
		MaxConnLifetime: EnvDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour),
		ConnectTimeout:  EnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
		QueryTimeout:    EnvDuration("DB_QUERY_TIMEOUT", 5*time.Second),
	}

	valkeyCfg := sharedvalkey.ValkeyConfig{
		DialTimeout:  EnvDuration("VALKEY_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  EnvDuration("VALKEY_READ_TIMEOUT", 2*time.Second),
		WriteTimeout: EnvDuration("VALKEY_WRITE_TIMEOUT", 2*time.Second),
		QueryTimeout: EnvDuration("VALKEY_QUERY_TIMEOUT", 2*time.Second),
	}

	// 2. サーキットブレイカーの構築
	dbBreaker := resilience.NewCircuitBreaker(resilience.Settings{ //nolint:contextcheck // Global breaker doesn't rely on request context
		Name:             "postgres",
		FailureThreshold: envFloat("DB_BREAKER_FAILURE_THRESHOLD", 0.5),
		Cooldown:         EnvDuration("DB_BREAKER_COOLDOWN", 30*time.Second),
		MinRequests:      envCircuitBreakerMinRequests("DB_BREAKER_MIN_REQUESTS"),
	})

	valkeyBreaker := resilience.NewCircuitBreaker(resilience.Settings{ //nolint:contextcheck // Global breaker doesn't rely on request context
		Name:             "valkey",
		FailureThreshold: envFloat("VALKEY_BREAKER_FAILURE_THRESHOLD", 0.5),
		Cooldown:         EnvDuration("VALKEY_BREAKER_COOLDOWN", 15*time.Second),
		MinRequests:      envCircuitBreakerMinRequests("VALKEY_BREAKER_MIN_REQUESTS"),
	})

	// 3. 接続オープン
	pool, err := postgres.Open(ctx, databaseURL, dbCfg)
	if err != nil {
		return nil, err
	}
	resilientDB := postgres.NewResilientDB(pool, dbBreaker, dbCfg.QueryTimeout)
	tenantRepo := &tenancypostgres.TenantRepository{Pool: resilientDB}
	// NewKeyStore bootstraps the default tenant signing key, whose FK requires
	// the tenant row to exist first. Fresh databases (including `just dev`) must
	// establish this root aggregate before assembling dependent adapters.
	if err := tenantusecases.EnsureDefault(ctx, tenantRepo, time.Now().UTC()); err != nil {
		pool.Close()
		return nil, err
	}

	valkeyClient, err := sharedvalkey.Open(ctx, valkeyURL, valkeyCfg, valkeyBreaker)
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
	switch EnvDefault("EVENT_SINK", "console") {
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
		Tenancy: tenancy.Module{
			TenantRepo:         tenantRepo,
			AttrSchemaRepo:     &idmpostgres.TenantUserAttributeSchemaRepository{Pool: resilientDB},
			BrandingRepo:       &tenancypostgres.TenantBrandingRepository{Pool: resilientDB},
			BrandingAssetStore: &tenancypostgres.TenantBrandingAssetStore{Pool: resilientDB},
		},
		IdentityManagement: identitymanagement.Module{
			UserRepo:  &idmpostgres.UserRepository{Pool: resilientDB},
			GroupRepo: &idmpostgres.GroupRepository{Pool: resilientDB},
			AgentRepo: &idmpostgres.AgentRepository{Pool: resilientDB},
		},
		Authentication: authentication.Module{
			MfaFactorRepo:           &authnpostgres.MfaFactorRepository{Pool: resilientDB},
			MfaEnrollmentBypassRepo: &authnpostgres.MfaEnrollmentBypassRepository{Pool: resilientDB},
			PasswordHistoryRepo:     &authnpostgres.PasswordHistoryRepository{Pool: resilientDB},
			PasswordResetTokenStore: &authnpostgres.PasswordResetTokenStore{Pool: resilientDB},
			EmailChangeTokenStore:   &authnpostgres.EmailChangeTokenStore{Pool: resilientDB},
			SessionStore:            &authnvalkey.SessionStore{Client: valkeyClient},
			WebAuthnCredentialRepo:  &authnpostgres.WebAuthnCredentialRepository{Pool: resilientDB},
			WebAuthnSessionStore:    &authnvalkey.WebAuthnSessionStore{Client: valkeyClient},
			RecoveryCodeRepo:        &authnpostgres.RecoveryCodeRepository{Pool: resilientDB},
			NewLoginAttemptThrottle: func(configs authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle {
				return &authnvalkey.LoginAttemptThrottle{Client: valkeyClient, Configs: configs}
			},
			AuthEventBucketStore: &authnpostgres.AuthEventBucketStore{Pool: resilientDB},
		},
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
			EventSink:                  sink,
		},
		KeyStore: selectKeyStore(keyStore),
		Audit: audit.Module{
			AuditEventRepo:  &auditpostgres.AuditEventRepository{Pool: resilientDB},
			TenantSaltStore: postgres.NewTenantSaltStore(resilientDB),
		},
		WsFederation: wsfederation.Module{RPRepo: &wsfedpostgres.WsFedRelyingPartyRepository{Pool: resilientDB}},
		Saml:         saml.Module{SPRepo: &samlpostgres.SamlServiceProviderRepository{Pool: resilientDB}},
		Scim:         scim.Module{Repo: &scimpostgres.ScimRepository{Pool: resilientDB}},
		Jobs:         jobs.Module{Repo: &jobspostgres.JobRepository{Pool: resilientDB}},
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
