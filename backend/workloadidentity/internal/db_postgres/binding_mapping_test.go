package db_postgres

import (
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/testing_contract"
)

//spec:covers REQ-WORKLOADIDENTITY-013: 状態変更と対応付けの更新は互いを上書きせず、テナント境界と一意性を保つ。
func TestBindingMappingContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	other := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	replacement := seedAgent(t, db, tenant.ID)
	bundle := newTrustBundle(t, tenant.ID)
	if err := (&WorkloadTrustBundleRepository{Pool: db}).Save(t.Context(), bundle); err != nil {
		t.Fatal(err)
	}
	testing_contract.RunBindingMapping(t, testing_contract.BindingMappingFixture{Repository: &AgentWorkloadBindingRepository{Pool: db}, TenantA: tenant.ID, TenantB: other.ID, BundleID: bundle.ID, BindingID: newUUID(t), OtherID: newUUID(t), AgentID: agent.ID, ReplacementAgentID: replacement.ID, Now: testClock()})
}
