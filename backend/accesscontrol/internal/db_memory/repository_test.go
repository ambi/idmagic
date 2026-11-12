package db_memory_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/accesscontrol/internal/db_memory"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/testing_contract"
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

//spec:covers REQ-AUTHORIZATION-018: 最新でないバージョンを基にした登録を拒否し、同じバージョンを基にした同時の登録は一つだけを受理する。
func TestAuthorizationModelRepositoryContract(t *testing.T) {
	testing_contract.RunAuthorizationModelRepositoryContract(t, newFixture)
}
