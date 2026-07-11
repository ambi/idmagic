package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/authentication"
	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/identitymanagement"
	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/scim"
	scimmemory "github.com/ambi/idmagic/backend/scim/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/adapters/persistence/memory"
)

func assembleMemory() (*Dependencies, error) {
	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		return nil, err
	}
	return &Dependencies{
		TenantRepo:     memory.NewTenantRepository(),
		AttrSchemaRepo: idmmemory.NewTenantUserAttributeSchemaRepository(),
		IdentityManagement: identitymanagement.Module{
			UserRepo:  idmmemory.NewUserRepository(),
			GroupRepo: idmmemory.NewGroupRepository(),
			AgentRepo: idmmemory.NewAgentRepository(),
		},
		Authentication: authentication.Module{
			MfaFactorRepo:           authnmemory.NewMfaFactorRepository(),
			PasswordHistoryRepo:     authnmemory.NewPasswordHistoryRepository(),
			PasswordResetTokenStore: memory.NewPasswordResetTokenStore(),
			EmailChangeTokenStore:   authnmemory.NewEmailChangeTokenStore(),
			SessionStore:            memory.NewSessionStore(),
			WebAuthnCredentialRepo:  authnmemory.NewWebAuthnCredentialRepository(),
			WebAuthnSessionStore:    memory.NewWebAuthnSessionStore(),
			RecoveryCodeRepo:        authnmemory.NewRecoveryCodeRepository(),
			NewLoginAttemptThrottle: func(configs authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle {
				return authnmemory.NewLoginAttemptThrottle(configs)
			},
			AuthEventBucketStore: authnmemory.NewAuthEventBucketStore(),
		},
		OAuth2: oauth2.Module{
			ClientRepo:                 oauth2memory.NewClientRepository(),
			ConsentRepo:                oauth2memory.NewConsentRepository(),
			AuthzDetailTypeRepo:        oauth2memory.NewAuthorizationDetailTypeRepository(),
			RequestStore:               memory.NewAuthorizationRequestStore(),
			CodeStore:                  memory.NewAuthorizationCodeStore(),
			PARStore:                   memory.NewPARStore(),
			RefreshStore:               oauth2memory.NewRefreshTokenStore(),
			DeviceCodeStore:            memory.NewDeviceCodeStore(),
			DpopReplayStore:            memory.NewDpopReplayStore(),
			ClientAssertionReplayStore: memory.NewClientAssertionReplayStore(),
			AccessTokenDenylist:        memory.NewAccessTokenDenylist(),
			AuditEventRepo:             oauth2memory.NewAuditEventStore(0),
			EventSink:                  eventsink.NewConsoleSink(),
		},
		KeyStore:        selectKeyStore(oauthports.KeyStore(keyStore)),
		TenantSaltStore: crypto.NewInMemoryTenantSaltStore(),
		WsFederation:    wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		Saml:            saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		Scim:            scim.Module{Repo: scimmemory.NewScimRepository()},
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
