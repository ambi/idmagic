package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/signinpolicy"

	"github.com/ambi/idmagic/backend/apitoken"

	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"

	"github.com/ambi/idmagic/backend/accesscontrol"
	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
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
	tenancyModule := tenancy.NewMemoryModule()
	auditModule := audit.NewMemoryModule(0)
	idmModule := idmanagement.NewMemoryModule()
	authnModule := authentication.NewMemoryModule(idmModule.UserRepo)
	idmModule = idmModule.WithMemoryUserImport(authnModule.PasswordHistoryRepo, tenancyModule.QuotaRepo, auditModule.AuditEventRepo)
	userRepo := idmModule.UserRepo
	governanceModule := idgovernance.NewMemoryModule(userRepo)
	applicationModule := application.NewMemoryModule()
	assignmentRepo := applicationModule.AssignmentRepo
	provisioningModule := provisioning.NewMemoryModule()
	applicationModule.ProvisioningNotifier = provisioningModule.AssignmentNotifier(assignmentRepo)
	idmModule.UserMutationCommitter = governanceModule.UserMutationCommitter
	idmModule.ProvisioningNotifier = provisioningModule.UserNotifier(assignmentRepo)
	idmModule.GroupProvisioningNotifier = provisioningModule.GroupNotifier(assignmentRepo)
	// CIMD の解決のイベントの発行先は NewEmitFunc の後で cmd/idmagic/server.go が結ぶ。
	oauth2Module := oauth2.NewMemoryModule().WithClientMetadataDocuments()
	oauth2Module.EventSink = sinks_console.NewConsoleSink()
	return &Dependencies{
		Tenancy:          tenancyModule,
		IdManagement:     idmModule,
		IdGovernance:     governanceModule,
		Authentication:   authnModule,
		OAuth2:           oauth2Module,
		SigningKeys:      signingkeys.Module{KeyStore: selectKeyStore(cfg, keyStore)},
		DataKeys:         dataKeysModule,
		Audit:            auditModule,
		WsFederation:     wsfederation.NewMemoryModule(),
		Saml:             saml.NewMemoryModule(),
		Sourcing:         sourcing.NewMemoryModule(),
		ApiTokens:        apitoken.NewMemoryModule(),
		Jobs:             jobs.NewMemoryModule(),
		Application:      applicationModule,
		SignInPolicy:     signinpolicy.NewMemoryModule(),
		Provisioning:     provisioningModule,
		WorkloadIdentity: workloadidentity.NewMemoryModule(),
		SharedSignals:    sharedsignals.NewMemoryModule(),
		AccessControl:    accesscontrol.NewMemoryModule(),
		RateLimit: rlports.Module{
			NewRateLimiter: func(configs rlports.RateLimitConfigs) rlports.RateLimiter {
				return ratelimitmemory.NewRateLimiter(configs)
			},
		},
		Close:  func() {},
		DbPing: func(c context.Context) error { return nil },
	}, nil
}
