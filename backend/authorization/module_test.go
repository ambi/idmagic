package authorization_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/authorization"
	"github.com/ambi/idmagic/backend/authorization/internal/testing_contract"
)

// 組み立て地点はアダプターではなく NewMemoryModule を使うので、それが組む
// Repository の組に、アダプター単体と同じ契約を当てる。
func memoryModuleFixture(_ *testing.T, _ ...string) testing_contract.Fixture {
	module := authorization.NewMemoryModule()
	return testing_contract.Fixture{Tuples: module.TupleRepo, Models: module.ModelRepo}
}

func TestMemoryModuleSatisfiesRelationTupleRepositoryContract(t *testing.T) {
	testing_contract.RunRelationTupleRepositoryContract(t, memoryModuleFixture)
}

func TestMemoryModuleSatisfiesAuthorizationModelRepositoryContract(t *testing.T) {
	testing_contract.RunAuthorizationModelRepositoryContract(t, memoryModuleFixture)
}
