package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/ratelimit/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Limiter: &RateLimiter{Pool: db, Configs: ports.RateLimitConfigs{"contract": {MaxRequests: 2, WindowSeconds: 60}, "other-policy": {MaxRequests: 2, WindowSeconds: 60}}}, Context: tenantports.WithTenant(context.Background(), tenant, "", ""), Now: pgtest.Now()}
	})
}
