package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			DetailTypes: NewAuthorizationDetailTypeRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}

//spec:covers REQ-OAUTH2-072: メモリの保存先が許可の共有契約を満たすこと
func TestCrossAppAccessPolicyRepositoryContract(t *testing.T) {
	testing_contract.RunCrossAppAccess(t, testing_contract.CrossAppAccessFixture{
		Policies: NewCrossAppAccessPolicyRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
		Now: time.Now().UTC().Truncate(time.Microsecond),
	})
}

//spec:covers REQ-OAUTH2-077: メモリの保存先がガードレールの共有契約を満たすこと
func TestAgentGuardrailPolicyRepositoryContract(t *testing.T) {
	testing_contract.RunAgentGuardrail(t, testing_contract.AgentGuardrailFixture{
		Repo: NewAgentGuardrailPolicyRepository(), Context: context.Background(),
		TenantID: "tenant-a", Other: "tenant-b", AgentID: "agent-a",
		Now: time.Now().UTC().Truncate(time.Microsecond),
	})
}
