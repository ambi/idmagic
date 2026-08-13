package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	"github.com/ambi/idmagic/backend/authentication"
	authnmemory "github.com/ambi/idmagic/backend/authentication/db_memory"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationsecrets "github.com/ambi/idmagic/backend/authentication/federation/secrets_env"
	mfamemory "github.com/ambi/idmagic/backend/authentication/mfa/db_memory"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeysmemory "github.com/ambi/idmagic/backend/datakeys/db_memory"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/idgovernance"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/db_memory"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	"github.com/ambi/idmagic/backend/oauth2"
	cimdhttp "github.com/ambi/idmagic/backend/oauth2/client/cimd_http"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/provisioning"
	provisioningmemory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	ratelimitmemory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/security/salts_memory"
	"github.com/ambi/idmagic/backend/sharedsignals"
	sharedsignalsmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/sourcing"
	scimmemory "github.com/ambi/idmagic/backend/sourcing/scim/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/workloadidentity"
	workloadidentitymemory "github.com/ambi/idmagic/backend/workloadidentity/db_memory"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"
)

func assembleMemory(cfg SharedConfig) (*Dependencies, error) {
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		return nil, err
	}
	masterKeyProvider, err := selectMasterKeyProvider(cfg)
	if err != nil {
		return nil, err
	}
	dataKeysRepo := datakeysmemory.NewDataKeyRepository()
	dataKeysCrypto := envelope_crypto.NewTinkEnvelopeCrypto(masterKeyProvider)
	dataKeysCache := datakeysusecases.NewDataKeyCache(dataKeysRepo, dataKeysCrypto)
	// No FieldMigrator to register: the memory runtime's MfaFactorRepository
	// (below) never encrypts, so there is nothing for the
	// data_key_reencryption job to migrate (dev/test only).
	dataKeysMigrators := datakeysusecases.NewMigratorRegistry()
	userRepo := usermemory.NewUserRepository()
	userCSVArtifacts := usermemory.NewUserCSVArtifactStore()
	quotaRepo := tenancymemory.NewQuotaRepository()
	passwordHistoryRepo := passwordmemory.NewPasswordHistoryRepository()
	auditEventRepo := auditmemory.NewAuditEventStore(0)
	userImportCommitter := usermemory.UserImportRowCommitter{
		Users: userRepo, PasswordHistory: passwordHistoryRepo, Quota: quotaRepo, Audit: auditEventRepo,
	}
	workflowRepo := igmemory.NewLifecycleWorkflowRepository()
	workflowRunRepo := igmemory.NewLifecycleWorkflowRunRepository()
	workflowCapture := &igmemory.UserWorkflowCapture{Users: userRepo, Runs: workflowRunRepo}
	userMutationCommitter := igusecases.UserMutationCommitter{
		WorkflowRepo: workflowRepo,
		Capture:      workflowCapture,
		UserRepo:     userRepo,
		RunRepo:      workflowRunRepo,
	}
	assignmentRepo := appmemory.NewApplicationAssignmentRepository()
	provisioningModule := provisioning.Module{
		ConnectionRepo: provisioningmemory.NewProvisioningConnectionRepository(),
		RemoteLinkRepo: provisioningmemory.NewRemoteResourceLinkRepository(),
		DeliveryRepo:   provisioningmemory.NewProvisioningDeliveryRepository(),
	}
	federationRepos := federationmemory.NewRepositories()
	return &Dependencies{
		Tenancy: tenancy.Module{
			TenantRepo:            tenancymemory.NewTenantRepository(),
			AttrSchemaRepo:        usermemory.NewTenantUserAttributeSchemaRepository(),
			GroupAttrSchemaRepo:   groupmemory.NewTenantGroupAttributeSchemaRepository(),
			BrandingRepo:          tenancymemory.NewTenantBrandingRepository(),
			BrandingAssetStore:    tenancymemory.NewTenantBrandingAssetStore(),
			NotificationTemplates: tenancymemory.NewNotificationTemplateRepository(),
			QuotaRepo:             quotaRepo,
		},
		IdManagement: idmanagement.Module{
			UserRepo:              userRepo,
			GroupRepo:             groupmemory.NewGroupRepository(),
			AgentRepo:             agentmemory.NewAgentRepository(),
			EmailChangeTokenStore: usermemory.NewEmailChangeTokenStore(),
			UserCSVArtifacts:      userCSVArtifacts,
			UserImportCommitter:   userImportCommitter,
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
			FederationConnectionRepo: federationRepos.Connections,
			FederationIdentityRepo:   federationRepos.Identities,
			FederationAttemptStore:   federationRepos.Attempts,
			FederationReplayStore:    federationRepos.Replay,
			FederationSecretResolver: federationsecrets.Resolver{},
			MfaFactorRepo:            totpmemory.NewMfaFactorRepository(),
			MfaEnrollmentBypassRepo:  mfamemory.NewMfaEnrollmentBypassRepository(),
			PasswordHistoryRepo:      passwordHistoryRepo,
			PasswordResetTokenStore:  passwordmemory.NewPasswordResetTokenStore(),
			SessionStore:             sessionmemory.NewSessionStore(),
			WebAuthnCredentialRepo:   webauthnmemory.NewWebAuthnCredentialRepository(),
			WebAuthnSessionStore:     webauthnmemory.NewWebAuthnSessionStore(),
			RecoveryCodeRepo:         recoverymemory.NewRecoveryCodeRepository(),
			NewLoginAttemptThrottle: func(configs sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle {
				return sessionmemory.NewLoginAttemptThrottle(configs)
			},
			AuthEventBucketStore: authnmemory.NewAuthEventBucketStore(),
		},
		OAuth2: oauth2.Module{
			// ClientRepo は CIMD (Client ID Metadata Document) 解決をデコレータで足す。
			// 登録済みクライアントの挙動は完全にそのまま (FindByID がヒットすればデコレータは委譲する
			// だけ)。Emit は起動シーケンス上 NewEmitFunc より前にしか組み立てられないため、
			// cmd/idmagic/server.go 側で deps.OAuth2.ClientRepo を型アサートして事後設定する
			// (sessionManager.Emit と同じ配線パターン)。
			ClientRepo: &cimdhttp.ClientRepositoryWithCIMD{
				OAuth2ClientRepository: oauth2memory.NewClientRepository(),
				Fetcher:                cimdhttp.NewFetcher(),
			},
			ConsentRepo:                oauth2memory.NewConsentRepository(),
			AuthzDetailTypeRepo:        oauth2memory.NewAuthorizationDetailTypeRepository(),
			McpResourceServerRepo:      oauth2memory.NewMcpResourceServerRepository(),
			RequestStore:               oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:                  oauth2memory.NewAuthorizationCodeStore(),
			PARStore:                   oauth2memory.NewPARStore(),
			RefreshStore:               oauth2memory.NewRefreshTokenStore(),
			DeviceCodeStore:            oauth2memory.NewDeviceCodeStore(),
			ApprovalRequestStore:       oauth2memory.NewApprovalRequestStore(),
			DpopReplayStore:            oauth2memory.NewDpopReplayStore(),
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
			AccessTokenDenylist:        oauth2memory.NewAccessTokenDenylist(),
			EventSink:                  sinks_console.NewConsoleSink(),
		},
		SigningKeys: signingkeys.Module{KeyStore: selectKeyStore(cfg, keyStore)},
		DataKeys:    datakeys.Module{Repository: dataKeysRepo, Cache: dataKeysCache, Crypto: dataKeysCrypto, Migrators: dataKeysMigrators},
		Audit: audit.Module{
			AuditEventRepo:  auditEventRepo,
			TenantSaltStore: salts_memory.NewInMemoryTenantSaltStore(),
		},
		WsFederation: wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		Saml: func() saml.Module {
			repo := samlmemory.NewSamlServiceProviderRepository()
			return saml.Module{SPRepo: repo, ProfileRepo: repo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()}
		}(),
		Sourcing:  sourcing.Module{ScimRepo: scimmemory.NewScimRepository()},
		ApiTokens: apitoken.Module{Repo: apitokenmemory.NewRepository()},
		Jobs:      jobs.Module{Repo: jobsmemory.NewJobRepository()},
		Application: application.Module{
			Repo:                    appmemory.NewApplicationRepository(),
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          assignmentRepo,
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			SignInPolicyRepo:        appmemory.NewSignInPolicyRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
			ProvisioningNotifier:    provisioningModule.AssignmentNotifier(assignmentRepo),
		},
		Provisioning: provisioningModule,
		WorkloadIdentity: workloadidentity.Module{
			TrustBundleRepo: workloadidentitymemory.NewWorkloadTrustBundleRepository(),
			BindingRepo:     workloadidentitymemory.NewAgentWorkloadBindingRepository(),
		},
		SharedSignals: sharedsignals.Module{
			RevocationEpochRepo:   sharedsignalsmemory.NewAgentRevocationEpochRepository(),
			StreamRepo:            sharedsignalsmemory.NewSsfStreamRepository(),
			TransmitterConfigRepo: sharedsignalsmemory.NewSsfTransmitterConfigRepository(),
			ReceiverConfigRepo:    sharedsignalsmemory.NewSsfReceiverConfigRepository(),
			DeliveryRepo:          sharedsignalsmemory.NewSecurityEventDeliveryRepository(),
			ReceivedEventRepo:     sharedsignalsmemory.NewReceivedSecurityEventRepository(),
		},
		RateLimit: rlports.Module{
			NewRateLimiter: func(configs rlports.RateLimitConfigs) rlports.RateLimiter {
				return ratelimitmemory.NewRateLimiter(configs)
			},
		},
		Close:  func() {},
		DbPing: func(c context.Context) error { return nil },
	}, nil
}
