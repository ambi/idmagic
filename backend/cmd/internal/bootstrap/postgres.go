package bootstrap

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"

	"github.com/ambi/idmagic/backend/application"
	apppostgres "github.com/ambi/idmagic/backend/application/db_postgres"
	"github.com/ambi/idmagic/backend/audit"

	"github.com/ambi/idmagic/backend/authentication"
	authnpostgres "github.com/ambi/idmagic/backend/authentication/db_postgres"
	federationpostgres "github.com/ambi/idmagic/backend/authentication/federation/db_postgres"
	federationsecrets "github.com/ambi/idmagic/backend/authentication/federation/secrets_env"
	mfapostgres "github.com/ambi/idmagic/backend/authentication/mfa/db_postgres"
	passwordpostgres "github.com/ambi/idmagic/backend/authentication/password/db_postgres"
	recoverypostgres "github.com/ambi/idmagic/backend/authentication/recovery/db_postgres"
	securitynotificationpostgres "github.com/ambi/idmagic/backend/authentication/securitynotification/db_postgres"
	sessionpostgres "github.com/ambi/idmagic/backend/authentication/session/db_postgres"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	totppostgres "github.com/ambi/idmagic/backend/authentication/totp/db_postgres"
	trusteddevicepostgres "github.com/ambi/idmagic/backend/authentication/trusteddevice/db_postgres"
	webauthnpostgres "github.com/ambi/idmagic/backend/authentication/webauthn/db_postgres"
	"github.com/ambi/idmagic/backend/authorization"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentpostgres "github.com/ambi/idmagic/backend/idmanagement/agent/db_postgres"
	idmpostgres "github.com/ambi/idmagic/backend/idmanagement/db_postgres"
	grouppostgres "github.com/ambi/idmagic/backend/idmanagement/group/db_postgres"
	userpostgres "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	"github.com/ambi/idmagic/backend/jobs"
	jobspostgres "github.com/ambi/idmagic/backend/jobs/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2"
	cimdhttp "github.com/ambi/idmagic/backend/oauth2/client/cimd_http"
	oauth2clientpostgres "github.com/ambi/idmagic/backend/oauth2/client/db_postgres"
	oauth2consentpostgres "github.com/ambi/idmagic/backend/oauth2/consent/db_postgres"
	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	logoutpostgres "github.com/ambi/idmagic/backend/oauth2/logout/db_postgres"
	oauth2tokenpostgres "github.com/ambi/idmagic/backend/oauth2/token/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning"

	"github.com/ambi/idmagic/backend/saml"

	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	ratelimitpostgres "github.com/ambi/idmagic/backend/shared/ratelimit/db_postgres"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/resilience"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"

	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sharedsignals"
	"github.com/ambi/idmagic/backend/signingkeys"

	"github.com/ambi/idmagic/backend/sourcing"

	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
	"github.com/ambi/idmagic/backend/workloadidentity"

	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedpostgres "github.com/ambi/idmagic/backend/wsfederation/db_postgres"
)

// assemblePostgres は PostgreSQL 単一依存の構成を組み立てる。durable state と
// 揮発性の認証 / OAuth2 一時状態の双方を PostgreSQL に載せる。cfg は呼び出し側が
// LoadSharedConfig + ConfigLoader.Err() で既に検証済み (DATABASE_URL 必須等) である
// ことを前提とする。
func assemblePostgres(ctx context.Context, cfg SharedConfig) (*Dependencies, error) {
	// 1. サーキットブレイカーの構築
	dbBreaker := resilience.NewCircuitBreaker(cfg.DBBreaker) //nolint:contextcheck // Global breaker doesn't rely on request context

	// 2. 接続オープン
	pool, err := postgres.Open(ctx, cfg.DatabaseURL.Value(), cfg.DB)
	if err != nil {
		return nil, err
	}
	resilientDB := postgres.NewResilientDB(pool, dbBreaker, cfg.DB.QueryTimeout)
	tenantRepo := &tenancypostgres.TenantRepository{Pool: resilientDB}
	// NewKeyStore bootstraps the default tenant signing key, whose FK requires
	// the tenant row to exist first. Fresh databases (including `mise run dev`) must
	// establish this root aggregate before assembling dependent adapters.
	if err := tenantusecases.EnsureDefault(ctx, tenantRepo, time.Now().UTC()); err != nil {
		pool.Close()
		return nil, err
	}

	keyStore, err := signingkeys.NewKeyStore(ctx, resilientDB)
	if err != nil {
		pool.Close()
		return nil, err
	}

	masterKeyProvider, err := selectMasterKeyProvider(cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	dataKeysCrypto := envelope_crypto.NewTinkEnvelopeCrypto(masterKeyProvider)
	dataKeysModule, err := datakeys.NewPostgresModule(ctx, resilientDB, dataKeysCrypto)
	if err != nil {
		pool.Close()
		return nil, err
	}
	mfaSecretCipher := dataKeysModule.FieldCipher()
	mfaFactorRepo := &totppostgres.MfaFactorRepository{Pool: resilientDB, Cipher: mfaSecretCipher}
	federationConnectionRepo := &federationpostgres.ConnectionRepository{Pool: resilientDB, Cipher: mfaSecretCipher}
	// dataKeysMigrators feeds the data_key_reencryption job (wi-97 T006):
	// every owning module registers its FieldMigrator here so Rotate can
	// enqueue per-migrator backfill jobs and Destroy's gate can verify no
	// migrator still has pending rows before crypto-shredding a version.
	dataKeysMigrators := dataKeysModule.Migrators
	dataKeysMigrators.Register(totppostgres.MfaFactorMigratorName, &totppostgres.MfaFactorReencryptor{Repo: mfaFactorRepo})
	dataKeysMigrators.Register(federationpostgres.IdentityProviderSecretMigratorName, &federationpostgres.ConnectionSecretReencryptor{
		Repo: federationConnectionRepo, EnvResolver: federationsecrets.Resolver{},
	})

	userRepo := &userpostgres.UserRepository{Pool: resilientDB}
	csvArtifacts := &idmpostgres.CSVArtifactStore{Pool: resilientDB}
	userImportCommitter := userpostgres.NewUserImportRowCommitter(resilientDB, tenancypostgres.QuotaRepositoryInTx)
	groupImportCommitter := grouppostgres.NewGroupImportRowCommitter(resilientDB, tenancypostgres.QuotaRepositoryInTx)
	groupMembershipImportCommitter := grouppostgres.GroupMembershipImportRowCommitter{Pool: resilientDB}
	governanceModule := idgovernance.NewPostgresModule(resilientDB, userRepo)
	assignmentRepo := &apppostgres.ApplicationAssignmentRepository{Pool: resilientDB}
	provisioningModule := provisioning.NewPostgresModule(resilientDB)

	return &Dependencies{
		Tenancy: tenancy.Module{
			TenantRepo:            tenantRepo,
			AttrSchemaRepo:        &userpostgres.TenantUserAttributeSchemaRepository{Pool: resilientDB},
			GroupAttrSchemaRepo:   &grouppostgres.TenantGroupAttributeSchemaRepository{Pool: resilientDB},
			BrandingRepo:          &tenancypostgres.TenantBrandingRepository{Pool: resilientDB},
			BrandingAssetStore:    &tenancypostgres.TenantBrandingAssetStore{Pool: resilientDB},
			NotificationTemplates: &tenancypostgres.NotificationTemplateRepository{Pool: resilientDB},
			QuotaRepo:             tenancypostgres.NewQuotaRepository(resilientDB),
		},
		IdManagement: idmanagement.Module{
			UserRepo:                       userRepo,
			GroupRepo:                      &grouppostgres.GroupRepository{Pool: resilientDB},
			AgentRepo:                      &agentpostgres.AgentRepository{Pool: resilientDB},
			EmailChangeTokenStore:          &userpostgres.EmailChangeTokenStore{Pool: resilientDB},
			CSVArtifacts:                   csvArtifacts,
			UserImportCommitter:            userImportCommitter,
			GroupImportCommitter:           groupImportCommitter,
			GroupMembershipImportCommitter: groupMembershipImportCommitter,
			UserMutationCommitter:          governanceModule.UserMutationCommitter,
			ProvisioningNotifier:           provisioningModule.UserNotifier(assignmentRepo),
			GroupProvisioningNotifier:      provisioningModule.GroupNotifier(assignmentRepo),
		},
		IdGovernance: governanceModule,
		Authentication: authentication.Module{
			FederationConnectionRepo: federationConnectionRepo,
			FederationIdentityRepo:   &federationpostgres.IdentityRepository{Pool: resilientDB},
			FederationAttemptStore:   &federationpostgres.AttemptStore{Pool: resilientDB},
			FederationReplayStore:    &federationpostgres.ReplayStore{Pool: resilientDB},
			FederationSecretResolver: federationsecrets.Resolver{},
			MfaFactorRepo:            mfaFactorRepo,
			MfaEnrollmentBypassRepo:  &mfapostgres.MfaEnrollmentBypassRepository{Pool: resilientDB},
			PasswordHistoryRepo:      &passwordpostgres.PasswordHistoryRepository{Pool: resilientDB},
			PasswordResetTokenStore:  &passwordpostgres.PasswordResetTokenStore{Pool: resilientDB},
			SessionStore:             &sessionpostgres.SessionRepository{Pool: resilientDB},
			WebAuthnCredentialRepo:   &webauthnpostgres.WebAuthnCredentialRepository{Pool: resilientDB},
			WebAuthnSessionStore:     &webauthnpostgres.WebAuthnSessionStore{Pool: resilientDB},
			RecoveryCodeRepo:         &recoverypostgres.RecoveryCodeRepository{Pool: resilientDB},
			TrustedDeviceRepo:        &trusteddevicepostgres.TrustedDeviceRepository{Pool: resilientDB},

			NotificationPreferenceRepo: &securitynotificationpostgres.PreferenceRepository{Pool: resilientDB},
			KnownSignInDeviceRepo:      &securitynotificationpostgres.KnownDeviceRepository{Pool: resilientDB},
			NewLoginAttemptThrottle: func(configs sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle {
				return &sessionpostgres.LoginAttemptThrottle{Pool: resilientDB, Configs: configs}
			},
			AuthEventBucketStore: &authnpostgres.AuthEventBucketStore{Pool: resilientDB},
		},
		OAuth2: oauth2.Module{
			// See memory.go for why CIMD resolution is wired as a decorator here and
			// Emit is set post-hoc in cmd/idmagic/server.go.
			ClientRepo: &cimdhttp.ClientRepositoryWithCIMD{
				OAuth2ClientRepository: &oauth2clientpostgres.OAuth2ClientRepository{Pool: resilientDB},
				Fetcher:                cimdhttp.NewFetcher(),
			},
			ConsentRepo:                &oauth2consentpostgres.ConsentRepository{Pool: resilientDB},
			AuthzDetailTypeRepo:        &oauth2postgres.AuthorizationDetailTypeRepository{Pool: resilientDB},
			McpResourceServerRepo:      &oauth2postgres.McpResourceServerRepository{Pool: resilientDB},
			RequestStore:               &oauth2postgres.AuthorizationRequestStore{Pool: resilientDB},
			CodeStore:                  &oauth2postgres.AuthorizationCodeStore{Pool: resilientDB},
			PARStore:                   &oauth2postgres.PARStore{Pool: resilientDB},
			RefreshStore:               &oauth2tokenpostgres.RefreshTokenStore{Pool: resilientDB},
			DeviceCodeStore:            &oauth2postgres.DeviceCodeStore{Pool: resilientDB},
			ApprovalRequestStore:       &oauth2postgres.ApprovalRequestStore{Pool: resilientDB},
			DpopReplayStore:            &oauth2postgres.ReplayStore{Pool: resilientDB, Kind: "dpop"},
			ClientAssertionReplayStore: &oauth2postgres.ReplayStore{Pool: resilientDB, Kind: "client_assertion"},
			ClientSessionStore:         &logoutpostgres.ClientSessionStore{Pool: resilientDB},
			LogoutNotificationStore:    &logoutpostgres.NotificationStore{Pool: resilientDB},
			AccessTokenDenylist:        &oauth2postgres.AccessTokenDenylist{Pool: resilientDB},
			EventSink:                  sinks_console.NewConsoleSink(),
		},
		SigningKeys:  signingkeys.Module{KeyStore: selectKeyStore(cfg, keyStore)},
		DataKeys:     dataKeysModule,
		Audit:        audit.NewPostgresModule(resilientDB),
		WsFederation: wsfederation.Module{RPRepo: &wsfedpostgres.WsFedRelyingPartyRepository{Pool: resilientDB}},
		Saml:         saml.NewPostgresModule(resilientDB),
		Sourcing:     sourcing.NewPostgresModule(resilientDB),
		ApiTokens:    apitoken.NewPostgresModule(resilientDB),
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
		Provisioning:     provisioningModule,
		WorkloadIdentity: workloadidentity.NewPostgresModule(resilientDB),
		SharedSignals:    sharedsignals.NewPostgresModule(resilientDB),
		Authorization:    authorization.NewPostgresModule(resilientDB),
		RateLimit: rlports.Module{
			NewRateLimiter: func(configs rlports.RateLimitConfigs) rlports.RateLimiter {
				return &ratelimitpostgres.RateLimiter{Pool: resilientDB, Configs: configs}
			},
		},
		Close: func() {
			pool.Close()
		},
		DbPing: func(c context.Context) error {
			return pool.Ping(c)
		},
	}, nil
}
