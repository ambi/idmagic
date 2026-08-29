package bootstrap

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenpostgres "github.com/ambi/idmagic/backend/apitoken/db_postgres"
	"github.com/ambi/idmagic/backend/application"
	apppostgres "github.com/ambi/idmagic/backend/application/db_postgres"
	"github.com/ambi/idmagic/backend/audit"
	auditpostgres "github.com/ambi/idmagic/backend/audit/db_postgres"
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
	authorizationpostgres "github.com/ambi/idmagic/backend/authorization/db_postgres"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeyspostgres "github.com/ambi/idmagic/backend/datakeys/db_postgres"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/idgovernance"
	igpostgres "github.com/ambi/idmagic/backend/idgovernance/db_postgres"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
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
	oauth2tokenpostgres "github.com/ambi/idmagic/backend/oauth2/token/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning"
	provisioningpostgres "github.com/ambi/idmagic/backend/provisioning/db_postgres"
	"github.com/ambi/idmagic/backend/saml"
	samlpostgres "github.com/ambi/idmagic/backend/saml/db_postgres"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	ratelimitpostgres "github.com/ambi/idmagic/backend/shared/ratelimit/db_postgres"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/resilience"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sharedsignals"
	sharedsignalspostgres "github.com/ambi/idmagic/backend/sharedsignals/db_postgres"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingpostgres "github.com/ambi/idmagic/backend/signingkeys/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing"
	scimpostgres "github.com/ambi/idmagic/backend/sourcing/scim/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
	"github.com/ambi/idmagic/backend/workloadidentity"
	workloadidentitypostgres "github.com/ambi/idmagic/backend/workloadidentity/db_postgres"
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

	keyStore, err := signingpostgres.NewKeyStore(ctx, resilientDB)
	if err != nil {
		pool.Close()
		return nil, err
	}

	dataKeysRepo, err := datakeyspostgres.NewDataKeyRepository(ctx, resilientDB)
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
	dataKeysCache := datakeysusecases.NewDataKeyCache(dataKeysRepo, dataKeysCrypto)
	mfaSecretCipher := &datakeys.FieldCipher{Repository: dataKeysRepo, Cache: dataKeysCache, Crypto: dataKeysCrypto}
	mfaFactorRepo := &totppostgres.MfaFactorRepository{Pool: resilientDB, Cipher: mfaSecretCipher}
	federationConnectionRepo := &federationpostgres.ConnectionRepository{Pool: resilientDB, Cipher: mfaSecretCipher}
	// dataKeysMigrators feeds the data_key_reencryption job (wi-97 T006):
	// every owning context registers its FieldMigrator here so Rotate can
	// enqueue per-migrator backfill jobs and Destroy's gate can verify no
	// migrator still has pending rows before crypto-shredding a version.
	dataKeysMigrators := datakeysusecases.NewMigratorRegistry()
	dataKeysMigrators.Register(totppostgres.MfaFactorMigratorName, &totppostgres.MfaFactorReencryptor{Repo: mfaFactorRepo})
	dataKeysMigrators.Register(federationpostgres.IdentityProviderSecretMigratorName, &federationpostgres.ConnectionSecretReencryptor{
		Repo: federationConnectionRepo, EnvResolver: federationsecrets.Resolver{},
	})

	userRepo := &userpostgres.UserRepository{Pool: resilientDB}
	csvArtifacts := &idmpostgres.CSVArtifactStore{Pool: resilientDB}
	userImportCommitter := userpostgres.UserImportRowCommitter{Pool: resilientDB}
	groupImportCommitter := grouppostgres.GroupImportRowCommitter{Pool: resilientDB}
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
			TenantRepo:            tenantRepo,
			AttrSchemaRepo:        &userpostgres.TenantUserAttributeSchemaRepository{Pool: resilientDB},
			GroupAttrSchemaRepo:   &grouppostgres.TenantGroupAttributeSchemaRepository{Pool: resilientDB},
			BrandingRepo:          &tenancypostgres.TenantBrandingRepository{Pool: resilientDB},
			BrandingAssetStore:    &tenancypostgres.TenantBrandingAssetStore{Pool: resilientDB},
			NotificationTemplates: &tenancypostgres.NotificationTemplateRepository{Pool: resilientDB},
			QuotaRepo:             tenancypostgres.NewQuotaRepository(resilientDB),
		},
		IdManagement: idmanagement.Module{
			UserRepo:              userRepo,
			GroupRepo:             &grouppostgres.GroupRepository{Pool: resilientDB},
			AgentRepo:             &agentpostgres.AgentRepository{Pool: resilientDB},
			EmailChangeTokenStore: &userpostgres.EmailChangeTokenStore{Pool: resilientDB},
			CSVArtifacts:          csvArtifacts,
			UserImportCommitter:   userImportCommitter,
			GroupImportCommitter:  groupImportCommitter,
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
			AccessTokenDenylist:        &oauth2postgres.AccessTokenDenylist{Pool: resilientDB},
			EventSink:                  sinks_console.NewConsoleSink(),
		},
		SigningKeys: signingkeys.Module{KeyStore: selectKeyStore(cfg, keyStore)},
		DataKeys:    datakeys.Module{Repository: dataKeysRepo, Cache: dataKeysCache, Crypto: dataKeysCrypto, Migrators: dataKeysMigrators},
		Audit: audit.Module{
			AuditEventRepo:  &auditpostgres.AuditEventRepository{Pool: resilientDB},
			TenantSaltStore: postgres.NewTenantSaltStore(resilientDB),
		},
		WsFederation: wsfederation.Module{RPRepo: &wsfedpostgres.WsFedRelyingPartyRepository{Pool: resilientDB}},
		Saml: saml.Module{
			SPRepo:      &samlpostgres.SamlServiceProviderRepository{Pool: resilientDB},
			ProfileRepo: &samlpostgres.SamlIdentityProviderProfileRepository{Pool: resilientDB},
			ReplayStore: &samlpostgres.AuthnRequestReplayStore{Pool: resilientDB},
		},
		Sourcing:  sourcing.Module{ScimRepo: &scimpostgres.ScimRepository{Pool: resilientDB}},
		ApiTokens: apitoken.Module{Repo: &apitokenpostgres.Repository{Pool: resilientDB}},
		Jobs:      jobs.Module{Repo: &jobspostgres.JobRepository{Pool: resilientDB}},
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
		WorkloadIdentity: workloadidentity.Module{
			TrustBundleRepo: &workloadidentitypostgres.WorkloadTrustBundleRepository{Pool: resilientDB},
			BindingRepo:     &workloadidentitypostgres.AgentWorkloadBindingRepository{Pool: resilientDB},
		},
		SharedSignals: sharedsignals.Module{
			RevocationEpochRepo:   &sharedsignalspostgres.AgentRevocationEpochRepository{Pool: resilientDB},
			StreamRepo:            &sharedsignalspostgres.SsfStreamRepository{Pool: resilientDB},
			TransmitterConfigRepo: &sharedsignalspostgres.SsfTransmitterConfigRepository{Pool: resilientDB},
			ReceiverConfigRepo:    &sharedsignalspostgres.SsfReceiverConfigRepository{Pool: resilientDB},
			DeliveryRepo:          &sharedsignalspostgres.SecurityEventDeliveryRepository{Pool: resilientDB},
			ReceivedEventRepo:     &sharedsignalspostgres.ReceivedSecurityEventRepository{Pool: resilientDB},
		},
		Authorization: authorization.Module{
			TupleRepo: &authorizationpostgres.RelationTupleRepository{Pool: resilientDB},
			ModelRepo: &authorizationpostgres.AuthorizationModelRepository{Pool: resilientDB},
		},
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
