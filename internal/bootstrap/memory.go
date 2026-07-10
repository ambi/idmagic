package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/internal/application"
	appmemory "github.com/ambi/idmagic/internal/application/adapters/persistence/memory"
	authnports "github.com/ambi/idmagic/internal/authentication/ports"
	oauthports "github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/adapters/crypto"
	"github.com/ambi/idmagic/internal/shared/adapters/eventsink"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
)

func assembleMemory() (*Dependencies, error) {
	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		return nil, err
	}
	return &Dependencies{
		ScimRepo:                memory.NewScimRepository(),
		TenantRepo:              memory.NewTenantRepository(),
		AttrSchemaRepo:          memory.NewTenantUserAttributeSchemaRepository(),
		ClientRepo:              memory.NewClientRepository(),
		UserRepo:                memory.NewUserRepository(),
		GroupRepo:               memory.NewGroupRepository(),
		AgentRepo:               memory.NewAgentRepository(),
		MfaFactorRepo:           memory.NewMfaFactorRepository(),
		PasswordHistoryRepo:     memory.NewPasswordHistoryRepository(),
		PasswordResetTokenStore: memory.NewPasswordResetTokenStore(),
		EmailChangeTokenStore:   memory.NewEmailChangeTokenStore(),
		ConsentRepo:             memory.NewConsentRepository(),
		AuthzDetailTypeRepo:     memory.NewAuthorizationDetailTypeRepository(),
		RequestStore:            memory.NewAuthorizationRequestStore(),
		CodeStore:               memory.NewAuthorizationCodeStore(),
		PARStore:                memory.NewPARStore(),
		RefreshStore:            memory.NewRefreshTokenStore(),
		DeviceCodeStore:         memory.NewDeviceCodeStore(),
		DpopReplay:              memory.NewDpopReplayStore(),
		ClientAssertionReplay:   memory.NewClientAssertionReplayStore(),
		AccessTokenDenylist:     memory.NewAccessTokenDenylist(),
		SessionStore:            memory.NewSessionStore(),
		WebAuthnCredentialRepo:  memory.NewWebAuthnCredentialRepository(),
		WebAuthnSessionStore:    memory.NewWebAuthnSessionStore(),
		RecoveryCodeRepo:        memory.NewRecoveryCodeRepository(),
		NewLoginAttemptThrottle: func(configs authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle {
			return memory.NewLoginAttemptThrottle(configs)
		},
		KeyStore:             selectKeyStore(oauthports.KeyStore(keyStore)),
		TenantSaltStore:      crypto.NewInMemoryTenantSaltStore(),
		EventSink:            eventsink.NewConsoleSink(),
		AuditEventRepo:       memory.NewAuditEventStore(0),
		AuthEventBucketStore: memory.NewAuthEventBucketStore(),
		WsFedRPRepo:          memory.NewWsFedRelyingPartyRepository(),
		SamlSPRepo:           memory.NewSamlServiceProviderRepository(),
		Application: application.Module{
			Repo:                    appmemory.NewApplicationRepository(),
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          appmemory.NewApplicationAssignmentRepository(),
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			SignInPolicyRepo:        appmemory.NewSignInPolicyRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Close:      func() {},
		DbPing:     func(c context.Context) error { return nil },
		ValkeyPing: func(c context.Context) error { return nil },
	}, nil
}
