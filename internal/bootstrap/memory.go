package bootstrap

import (
	"context"

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
		NewLoginAttemptThrottle: func(configs authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle {
			return memory.NewLoginAttemptThrottle(configs)
		},
		KeyStore:                    selectKeyStore(oauthports.KeyStore(keyStore)),
		EventSink:                   eventsink.NewConsoleSink(),
		AuditEventRepo:              memory.NewAuditEventStore(0),
		AuthEventBucketStore:        memory.NewAuthEventBucketStore(),
		WsFedRPRepo:                 memory.NewWsFedRelyingPartyRepository(),
		SamlSPRepo:                  memory.NewSamlServiceProviderRepository(),
		ApplicationRepo:             memory.NewApplicationRepository(),
		ApplicationIconStore:        memory.NewApplicationIconStore(),
		ApplicationAssignmentRepo:   memory.NewApplicationAssignmentRepository(),
		ApplicationOrderingRepo:     memory.NewApplicationOrderingRepository(),
		ApplicationCategoryRepo:     memory.NewApplicationCategoryRepository(),
		ApplicationSignInPolicyRepo: memory.NewSignInPolicyRepository(),
		Close:                       func() {},
		DbPing:                      func(c context.Context) error { return nil },
		ValkeyPing:                  func(c context.Context) error { return nil },
	}, nil
}
