package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/apitoken"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	"github.com/ambi/idmagic/backend/audit"

	"github.com/ambi/idmagic/backend/authentication"
	authnmemory "github.com/ambi/idmagic/backend/authentication/db_memory"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationsecrets "github.com/ambi/idmagic/backend/authentication/federation/secrets_env"
	mfamemory "github.com/ambi/idmagic/backend/authentication/mfa/db_memory"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	securitynotificationmemory "github.com/ambi/idmagic/backend/authentication/securitynotification/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/trusteddevice/db_memory"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	"github.com/ambi/idmagic/backend/authorization"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	cimdhttp "github.com/ambi/idmagic/backend/oauth2/client/cimd_http"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	logoutmemory "github.com/ambi/idmagic/backend/oauth2/logout/db_memory"
	"github.com/ambi/idmagic/backend/provisioning"

	"github.com/ambi/idmagic/backend/saml"

	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	ratelimitmemory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"

	"github.com/ambi/idmagic/backend/sharedsignals"

	"github.com/ambi/idmagic/backend/signingkeys"

	"github.com/ambi/idmagic/backend/sourcing"

	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/workloadidentity"

	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"
)

func assembleMemory(cfg SharedConfig) (*Dependencies, error) {
	keyStore, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		return nil, err
	}
	masterKeyProvider, err := selectMasterKeyProvider(cfg)
	if err != nil {
		return nil, err
	}
	dataKeysCrypto := envelope_crypto.NewTinkEnvelopeCrypto(masterKeyProvider)
	dataKeysModule := datakeys.NewMemoryModule(dataKeysCrypto)
	// No FieldMigrator to register: the memory runtime's MfaFactorRepository
	// (below) never encrypts, so there is nothing for the
	// data_key_reencryption job to migrate (dev/test only).
	userRepo := usermemory.NewUserRepository()
	csvArtifacts := idmmemory.NewCSVArtifactStore()
	tenancyModule := tenancy.NewMemoryModule()
	tenancyModule.AttrSchemaRepo = usermemory.NewTenantUserAttributeSchemaRepository()
	tenancyModule.GroupAttrSchemaRepo = groupmemory.NewTenantGroupAttributeSchemaRepository()
	quotaRepo := tenancyModule.QuotaRepo
	passwordHistoryRepo := passwordmemory.NewPasswordHistoryRepository()
	auditModule := audit.NewMemoryModule(0)
	auditEventRepo := auditModule.AuditEventRepo
	groupRepo := groupmemory.NewGroupRepository()
	groupImportCommitter := groupmemory.NewGroupImportRowCommitter(groupRepo)
	groupMembershipImportCommitter := groupmemory.NewGroupMembershipImportRowCommitter(groupRepo)
	userImportCommitter := usermemory.UserImportRowCommitter{
		Users: userRepo, PasswordHistory: passwordHistoryRepo, Quota: quotaRepo, Audit: auditEventRepo,
	}
	governanceModule := idgovernance.NewMemoryModule(userRepo)
	assignmentRepo := appmemory.NewApplicationAssignmentRepository()
	provisioningModule := provisioning.NewMemoryModule()
	federationRepos := federationmemory.NewRepositories()
	return &Dependencies{
		Tenancy: tenancyModule,
		IdManagement: idmanagement.Module{
			UserRepo:                       userRepo,
			GroupRepo:                      groupRepo,
			AgentRepo:                      agentmemory.NewAgentRepository(),
			EmailChangeTokenStore:          usermemory.NewEmailChangeTokenStore(userRepo),
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
			FederationConnectionRepo: federationRepos.Connections,
			FederationIdentityRepo:   federationRepos.Identities,
			FederationAttemptStore:   federationRepos.Attempts,
			FederationReplayStore:    federationRepos.Replay,
			FederationSecretResolver: federationsecrets.Resolver{},
			MfaFactorRepo:            totpmemory.NewMfaFactorRepository(),
			MfaEnrollmentBypassRepo:  mfamemory.NewMfaEnrollmentBypassRepository(),
			PasswordHistoryRepo:      passwordHistoryRepo,
			PasswordResetTokenStore:  passwordmemory.NewPasswordResetTokenStore(userRepo, passwordHistoryRepo),
			SessionStore:             sessionmemory.NewSessionStore(),
			WebAuthnCredentialRepo:   webauthnmemory.NewWebAuthnCredentialRepository(),
			WebAuthnSessionStore:     webauthnmemory.NewWebAuthnSessionStore(),
			RecoveryCodeRepo:         recoverymemory.NewRecoveryCodeRepository(),
			TrustedDeviceRepo:        trusteddevicememory.NewTrustedDeviceRepository(),

			NotificationPreferenceRepo: securitynotificationmemory.NewPreferenceRepository(),
			KnownSignInDeviceRepo:      securitynotificationmemory.NewKnownDeviceRepository(),
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
			ClientSessionStore:         logoutmemory.NewClientSessionStore(),
			LogoutNotificationStore:    logoutmemory.NewLogoutNotificationStore(),
			AccessTokenDenylist:        oauth2memory.NewAccessTokenDenylist(),
			EventSink:                  sinks_console.NewConsoleSink(),
		},
		SigningKeys:  signingkeys.Module{KeyStore: selectKeyStore(cfg, keyStore)},
		DataKeys:     dataKeysModule,
		Audit:        auditModule,
		WsFederation: wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		Saml:         saml.NewMemoryModule(),
		Sourcing:     sourcing.NewMemoryModule(),
		ApiTokens:    apitoken.NewMemoryModule(),
		Jobs:         jobs.NewMemoryModule(),
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
		Provisioning:     provisioningModule,
		WorkloadIdentity: workloadidentity.NewMemoryModule(),
		SharedSignals:    sharedsignals.NewMemoryModule(),
		Authorization:    authorization.NewMemoryModule(),
		RateLimit: rlports.Module{
			NewRateLimiter: func(configs rlports.RateLimitConfigs) rlports.RateLimiter {
				return ratelimitmemory.NewRateLimiter(configs)
			},
		},
		Close:  func() {},
		DbPing: func(c context.Context) error { return nil },
	}, nil
}
