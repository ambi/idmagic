package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/datakeys/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	repo, err := NewDataKeyRepository(context.Background(), db)
	if err != nil {
		t.Fatalf("NewDataKeyRepository: %v", err)
	}
	tenant := seedTenant(t, repo)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: repo, TenantID: tenant.ID, Now: pgtest.Now()}
	})
}
