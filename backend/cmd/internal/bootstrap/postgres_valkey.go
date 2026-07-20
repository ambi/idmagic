package bootstrap

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ambi/idmagic/backend/application"
	apppostgres "github.com/ambi/idmagic/backend/application/db_postgres"
	"github.com/ambi/idmagic/backend/audit"
	auditpostgres "github.com/ambi/idmagic/backend/audit/db_postgres"
	"github.com/ambi/idmagic/backend/authentication"
	authnpostgres "github.com/ambi/idmagic/backend/authentication/db_postgres"
	mfapostgres "github.com/ambi/idmagic/backend/authentication/mfa/db_postgres"
	passwordpostgres "github.com/ambi/idmagic/backend/authentication/password/db_postgres"
	recoverypostgres "github.com/ambi/idmagic/backend/authentication/recovery/db_postgres"
	sessionpostgres "github.com/ambi/idmagic/backend/authentication/session/db_postgres"
	sessionvalkey "github.com/ambi/idmagic/backend/authentication/session/db_valkey"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	totppostgres "github.com/ambi/idmagic/backend/authentication/totp/db_postgres"
	webauthnpostgres "github.com/ambi/idmagic/backend/authentication/webauthn/db_postgres"
	webauthnvalkey "github.com/ambi/idmagic/backend/authentication/webauthn/db_valkey"
	"github.com/ambi/idmagic/backend/idgovernance"
	igpostgres "github.com/ambi/idmagic/backend/idgovernance/db_postgres"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentpostgres "github.com/ambi/idmagic/backend/idmanagement/agent/db_postgres"
	grouppostgres "github.com/ambi/idmagic/backend/idmanagement/group/db_postgres"
	userpostgres "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	"github.com/ambi/idmagic/backend/jobs"
	jobspostgres "github.com/ambi/idmagic/backend/jobs/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2clientpostgres "github.com/ambi/idmagic/backend/oauth2/client/db_postgres"
	oauth2consentpostgres "github.com/ambi/idmagic/backend/oauth2/consent/db_postgres"
	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	oauth2valkey "github.com/ambi/idmagic/backend/oauth2/db_valkey"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	oauth2tokenpostgres "github.com/ambi/idmagic/backend/oauth2/token/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning"
	provisioningpostgres "github.com/ambi/idmagic/backend/provisioning/db_postgres"
	"github.com/ambi/idmagic/backend/saml"
	samlpostgres "github.com/ambi/idmagic/backend/saml/db_postgres"
	samlvalkey "github.com/ambi/idmagic/backend/saml/db_valkey"
	"github.com/ambi/idmagic/backend/scim"
	scimpostgres "github.com/ambi/idmagic/backend/scim/db_postgres"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/resilience"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	sharedvalkey "github.com/ambi/idmagic/backend/shared/storage/db_valkey"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingpostgres "github.com/ambi/idmagic/backend/signingkeys/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedpostgres "github.com/ambi/idmagic/backend/wsfederation/db_postgres"
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

	keyStore, err := signingpostgres.NewKeyStore(ctx, resilientDB)
	if err != nil {
		pool.Close()
		_ = valkeyClient.Close()
		return nil, err
	}

	var sink oauthports.EventSink
	switch EnvDefault("EVENT_SINK", "console") {
	case "console":
		sink = sinks_console.NewConsoleSink()
	case "outbox":
		sink = &oauth2postgres.OutboxEventSink{Pool: resilientDB}
	default:
		pool.Close()
		_ = valkeyClient.Close()
		return nil, errors.New("EVENT_SINK must be console or outbox")
	}

	userRepo := &userpostgres.UserRepository{Pool: resilientDB}
	workflowRepo := &igpostgres.LifecycleWorkflowRepository{Pool: resilientDB}
	workflowRunRepo := &igpostgres.LifecycleWorkflowRunRepository{Pool: resilientDB}
	workflowCapture := &igpostgres.UserWorkflowCapture{Pool: resilientDB}
	userMutationCommitter := igusecases.UserMutationCommitter{
		WorkflowRepo: workflowRepo, Capture: workflowCapture, UserRepo: userRepo, RunRepo: workflowRunRepo,
	}
	assignmentRepo := &apppostgres.ApplicationAssignmentRepository{Pool: resilientDB}
	provisioningModule := provisioning.Module{
		ConnectionRepo: &provisioningpostgres.ProvisioningConnectionRepository{Pool: resilientDB},
		RemoteLinkRepo: &provisioningpostgres.RemoteResourceLinkRepository{Pool: resilientDB},
		DeliveryRepo:   &provisioningpostgres.ProvisioningDeliveryRepository{Pool: resilientDB},
	}

	return &Dependencies{
		Tenancy: tenancy.Module{
			TenantRepo:         tenantRepo,
			AttrSchemaRepo:     &userpostgres.TenantUserAttributeSchemaRepository{Pool: resilientDB},
			BrandingRepo:       &tenancypostgres.TenantBrandingRepository{Pool: resilientDB},
			BrandingAssetStore: &tenancypostgres.TenantBrandingAssetStore{Pool: resilientDB},
		},
		IdManagement: idmanagement.Module{
			UserRepo:              userRepo,
			GroupRepo:             &grouppostgres.GroupRepository{Pool: resilientDB},
			AgentRepo:             &agentpostgres.AgentRepository{Pool: resilientDB},
			EmailChangeTokenStore: &userpostgres.EmailChangeTokenStore{Pool: resilientDB},
			UserMutationCommitter: userMutationCommitter,
			ProvisioningNotifier:  provisioningModule.UserNotifier(assignmentRepo),
		},
		IdGovernance: idgovernance.Module{
			LifecycleWorkflowRepo:    workflowRepo,
			LifecycleWorkflowRunRepo: workflowRunRepo,
			UserWorkflowCapture:      workflowCapture,
			UserMutationCommitter:    userMutationCommitter,
		},
		Authentication: authentication.Module{
			MfaFactorRepo:           &totppostgres.MfaFactorRepository{Pool: resilientDB},
			MfaEnrollmentBypassRepo: &mfapostgres.MfaEnrollmentBypassRepository{Pool: resilientDB},
			PasswordHistoryRepo:     &passwordpostgres.PasswordHistoryRepository{Pool: resilientDB},
			PasswordResetTokenStore: &passwordpostgres.PasswordResetTokenStore{Pool: resilientDB},
			SessionStore:            &sessionpostgres.SessionRepository{Pool: resilientDB},
			WebAuthnCredentialRepo:  &webauthnpostgres.WebAuthnCredentialRepository{Pool: resilientDB},
			WebAuthnSessionStore:    &webauthnvalkey.WebAuthnSessionStore{Client: valkeyClient},
			RecoveryCodeRepo:        &recoverypostgres.RecoveryCodeRepository{Pool: resilientDB},
			NewLoginAttemptThrottle: func(configs sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle {
				return &sessionvalkey.LoginAttemptThrottle{Client: valkeyClient, Configs: configs}
			},
			AuthEventBucketStore: &authnpostgres.AuthEventBucketStore{Pool: resilientDB},
		},
		OAuth2: oauth2.Module{
			ClientRepo:                 &oauth2clientpostgres.OAuth2ClientRepository{Pool: resilientDB},
			ConsentRepo:                &oauth2consentpostgres.ConsentRepository{Pool: resilientDB},
			AuthzDetailTypeRepo:        &oauth2postgres.AuthorizationDetailTypeRepository{Pool: resilientDB},
			McpResourceServerRepo:      &oauth2postgres.McpResourceServerRepository{Pool: resilientDB},
			RequestStore:               &oauth2valkey.AuthorizationRequestStore{Client: valkeyClient},
			CodeStore:                  &oauth2valkey.AuthorizationCodeStore{Client: valkeyClient},
			PARStore:                   &oauth2valkey.PARStore{Client: valkeyClient},
			RefreshStore:               &oauth2tokenpostgres.RefreshTokenStore{Pool: resilientDB},
			DeviceCodeStore:            &oauth2valkey.DeviceCodeStore{Client: valkeyClient},
			DpopReplayStore:            &oauth2valkey.ReplayStore{Client: valkeyClient, Prefix: "dpop_replay:"},
			ClientAssertionReplayStore: &oauth2valkey.ReplayStore{Client: valkeyClient, Prefix: "client_assertion:"},
			AccessTokenDenylist:        &oauth2valkey.AccessTokenDenylist{Client: valkeyClient},
			EventSink:                  sink,
		},
		SigningKeys: signingkeys.Module{KeyStore: selectKeyStore(keyStore)},
		Audit: audit.Module{
			AuditEventRepo:  &auditpostgres.AuditEventRepository{Pool: resilientDB},
			TenantSaltStore: postgres.NewTenantSaltStore(resilientDB),
		},
		WsFederation: wsfederation.Module{RPRepo: &wsfedpostgres.WsFedRelyingPartyRepository{Pool: resilientDB}},
		Saml:         saml.Module{SPRepo: &samlpostgres.SamlServiceProviderRepository{Pool: resilientDB}, ReplayStore: &samlvalkey.AuthnRequestReplayStore{Client: valkeyClient}},
		Scim:         scim.Module{Repo: &scimpostgres.ScimRepository{Pool: resilientDB}},
		Jobs:         jobs.Module{Repo: &jobspostgres.JobRepository{Pool: resilientDB}},
		Application: application.Module{
			Repo:                    &apppostgres.ApplicationRepository{Pool: resilientDB},
			IconStore:               &apppostgres.ApplicationIconStore{Pool: resilientDB},
			AssignmentRepo:          assignmentRepo,
			OrderingRepo:            &apppostgres.ApplicationOrderingRepository{Pool: resilientDB},
			CategoryRepo:            &apppostgres.ApplicationCategoryRepository{Pool: resilientDB},
			SignInPolicyRepo:        &apppostgres.SignInPolicyRepository{Pool: resilientDB},
			DefaultSignInPolicyRepo: &apppostgres.DefaultSignInPolicyRepository{Pool: resilientDB},
			ProvisioningNotifier:    provisioningModule.AssignmentNotifier(assignmentRepo),
		},
		Provisioning: provisioningModule,
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
