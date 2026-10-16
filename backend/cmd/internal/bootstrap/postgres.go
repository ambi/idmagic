package bootstrap

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"

	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/authorization"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
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
	tenancyModule := tenancy.NewPostgresModule(resilientDB)
	idmModule := idmanagement.NewPostgresModule(resilientDB, tenancy.QuotaRepositoryInTx)
	tenancyModule.AttrSchemaRepo = idmModule.AttrSchemaRepo
	tenancyModule.GroupAttrSchemaRepo = idmModule.GroupAttrSchemaRepo
	// NewKeyStore bootstraps the default tenant signing key, whose FK requires
	// the tenant row to exist first. Fresh databases (including `mise run dev`) must
	// establish this root aggregate before assembling dependent adapters.
	if err := tenancyModule.EnsureDefault(ctx, time.Now().UTC()); err != nil {
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
	authnModule := authentication.NewPostgresModule(resilientDB, dataKeysModule.FieldCipher(), idmanagement.UserWriterInTx)
	// dataKeysModule.Migrators feeds the data_key_reencryption job (wi-97 T006):
	// every owning module registers its FieldMigrator here so Rotate can
	// enqueue per-migrator backfill jobs and Destroy's gate can verify no
	// migrator still has pending rows before crypto-shredding a version.
	authnModule.RegisterFieldMigrators(func(name string, migrator authentication.FieldMigrator) {
		dataKeysModule.Migrators.Register(name, migrator)
	})

	userRepo := idmModule.UserRepo
	governanceModule := idgovernance.NewPostgresModule(resilientDB, userRepo, idmanagement.UserWriterInTx)
	applicationModule := application.NewPostgresModule(resilientDB)
	assignmentRepo := applicationModule.AssignmentRepo
	provisioningModule := provisioning.NewPostgresModule(resilientDB)
	// CIMD の解決のイベントの発行先は NewEmitFunc の後で cmd/idmagic/server.go が結ぶ。
	oauth2Module := oauth2.NewPostgresModule(resilientDB).WithClientMetadataDocuments()
	oauth2Module.EventSink = sinks_console.NewConsoleSink()
	applicationModule.ProvisioningNotifier = provisioningModule.AssignmentNotifier(assignmentRepo)
	idmModule.UserMutationCommitter = governanceModule.UserMutationCommitter
	idmModule.ProvisioningNotifier = provisioningModule.UserNotifier(assignmentRepo)
	idmModule.GroupProvisioningNotifier = provisioningModule.GroupNotifier(assignmentRepo)

	return &Dependencies{
		Tenancy:          tenancyModule,
		IdManagement:     idmModule,
		IdGovernance:     governanceModule,
		Authentication:   authnModule,
		OAuth2:           oauth2Module,
		SigningKeys:      signingkeys.Module{KeyStore: selectKeyStore(cfg, keyStore)},
		DataKeys:         dataKeysModule,
		Audit:            audit.NewPostgresModule(resilientDB),
		WsFederation:     wsfederation.Module{RPRepo: &wsfedpostgres.WsFedRelyingPartyRepository{Pool: resilientDB}},
		Saml:             saml.NewPostgresModule(resilientDB),
		Sourcing:         sourcing.NewPostgresModule(resilientDB),
		ApiTokens:        apitoken.NewPostgresModule(resilientDB),
		Jobs:             jobs.NewPostgresModule(resilientDB),
		Application:      applicationModule,
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
