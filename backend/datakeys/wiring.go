package datakeys

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/internal/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/internal/db_postgres"
	"github.com/ambi/idmagic/backend/datakeys/internal/domain"
	"github.com/ambi/idmagic/backend/datakeys/internal/handlers_http"
	"github.com/ambi/idmagic/backend/datakeys/internal/usecases"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

func NewMemoryModule(crypto envelope_crypto.EnvelopeCrypto) Module {
	repo := db_memory.NewDataKeyRepository()
	return Module{Repository: repo, Crypto: crypto, Cache: usecases.NewDataKeyCache(repo, crypto), Migrators: usecases.NewMigratorRegistry()}
}

func NewPostgresModule(ctx context.Context, pool postgres.DB, crypto envelope_crypto.EnvelopeCrypto) (Module, error) {
	repo, err := db_postgres.NewDataKeyRepository(ctx, pool)
	if err != nil {
		return Module{}, err
	}
	return Module{Repository: repo, Crypto: crypto, Cache: usecases.NewDataKeyCache(repo, crypto), Migrators: usecases.NewMigratorRegistry()}, nil
}

func (m Module) FieldCipher() *FieldCipher {
	return &FieldCipher{Repository: m.Repository, Cache: m.Cache, Crypto: m.Crypto}
}

func (m Module) BootstrapTenantDataKey(ctx context.Context, tenantID string, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	return usecases.BootstrapTenantDataKey(ctx, usecases.Deps{Repository: m.Repository, Crypto: m.Crypto, Cache: m.Cache}, tenantID, now)
}

func (m Module) RotateTenantDataKey(ctx context.Context, tenantID string, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	return usecases.RotateTenantDataKey(ctx, usecases.Deps{Repository: m.Repository, Crypto: m.Crypto, Cache: m.Cache}, tenantID, now)
}

func (m Module) ReencryptionHandler(jobs jobsports.Enqueuer) jobsports.Handler {
	return usecases.ReencryptionHandler(usecases.ReencryptDeps{Repository: m.Repository, Migrators: m.Migrators, Jobs: jobs})
}

func EnqueueReencryptionJob(ctx context.Context, jobs jobsports.Enqueuer, tenantID, name string, now time.Time) error {
	return usecases.EnqueueReencryptionJob(ctx, jobs, tenantID, name, now)
}

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	TenantRepo tenantports.TenantRepository
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{Deps: d.Deps, Authenticator: d.Authenticator, Repository: m.Repository, Crypto: m.Crypto, TenantRepo: d.TenantRepo})
}
