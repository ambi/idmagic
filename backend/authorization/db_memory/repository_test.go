package db_memory_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/authorization/db_memory"
	"github.com/ambi/idmagic/backend/authorization/testing_contract"
)

func newFixture(_ *testing.T, _ ...string) testing_contract.Fixture {
	store := db_memory.NewStore()
	return testing_contract.Fixture{
		Tuples: db_memory.NewRelationTupleRepository(store),
		Models: db_memory.NewAuthorizationModelRepository(store),
	}
}

func TestRelationTupleRepositoryContract(t *testing.T) {
	testing_contract.RunRelationTupleRepositoryContract(t, newFixture)
}

func TestAuthorizationModelRepositoryContract(t *testing.T) {
	testing_contract.RunAuthorizationModelRepositoryContract(t, newFixture)
}
