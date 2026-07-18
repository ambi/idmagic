package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/authentication"
	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/idgovernance"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/adapters/persistence/memory"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/scim"
	scimmemory "github.com/ambi/idmagic/backend/scim/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/adapters/crypto"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/adapters/persistence/memory"
)

func assembleMemory() (*Dependencies, error) {
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		return nil, err
	}
	userRepo := idmmemory.NewUserRepository()
	workflowRepo := igmemory.NewLifecycleWorkflowRepository()
	workflowRunRepo := igmemory.NewLifecycleWorkflowRunRepository()
	workflowCapture := &igmemory.UserWorkflowCapture{Users: userRepo, Runs: workflowRunRepo}
	userMutationCommitter := igusecases.UserMutationCommitter{
		WorkflowRepo: workflowRepo,
		Capture:      workflowCapture,
		UserRepo:     userRepo,
		RunRepo:      workflowRunRepo,
	}
	return &Dependencies{
		Tenancy: tenancy.Module{
			TenantRepo:         tenancymemory.NewTenantRepository(),
			AttrSchemaRepo:     idmmemory.NewTenantUserAttributeSchemaRepository(),
			BrandingRepo:       tenancymemory.NewTenantBrandingRepository(),
			BrandingAssetStore: tenancymemory.NewTenantBrandingAssetStore(),
		},
		IdManagement: idmanagement.Module{
			UserRepo:              userRepo,
			GroupRepo:             idmmemory.NewGroupRepository(),
			AgentRepo:             idmmemory.NewAgentRepository(),
			UserMutationCommitter: userMutationCommitter,
		},
		IdGovernance: idgovernance.Module{
			LifecycleWorkflowRepo:    workflowRepo,
			LifecycleWorkflowRunRepo: workflowRunRepo,
			UserWorkflowCapture:      workflowCapture,
			UserMutationCommitter:    userMutationCommitter,
		},
		Authentication: authentication.Module{
			MfaFactorRepo:           authnmemory.NewMfaFactorRepository(),
			MfaEnrollmentBypassRepo: authnmemory.NewMfaEnrollmentBypassRepository(),
			PasswordHistoryRepo:     authnmemory.NewPasswordHistoryRepository(),
			PasswordResetTokenStore: authnmemory.NewPasswordResetTokenStore(),
			EmailChangeTokenStore:   authnmemory.NewEmailChangeTokenStore(),
			SessionStore:            authnmemory.NewSessionStore(),
			WebAuthnCredentialRepo:  authnmemory.NewWebAuthnCredentialRepository(),
			WebAuthnSessionStore:    authnmemory.NewWebAuthnSessionStore(),
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
			RequestStore:               oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:                  oauth2memory.NewAuthorizationCodeStore(),
			PARStore:                   oauth2memory.NewPARStore(),
			RefreshStore:               oauth2memory.NewRefreshTokenStore(),
			DeviceCodeStore:            oauth2memory.NewDeviceCodeStore(),
			DpopReplayStore:            oauth2memory.NewDpopReplayStore(),
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
			AccessTokenDenylist:        oauth2memory.NewAccessTokenDenylist(),
			EventSink:                  eventsink.NewConsoleSink(),
		},
		SigningKeys: signingkeys.Module{KeyStore: selectKeyStore(keyStore)},
		Audit: audit.Module{
			AuditEventRepo:  auditmemory.NewAuditEventStore(0),
			TenantSaltStore: crypto.NewInMemoryTenantSaltStore(),
		},
		WsFederation: wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		Saml:         saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository(), ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		Scim:         scim.Module{Repo: scimmemory.NewScimRepository()},
		Jobs:         jobs.Module{Repo: jobsmemory.NewJobRepository()},
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
