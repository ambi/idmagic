package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/internal/testing_contract"
)

//spec:covers REQ-WORKLOADIDENTITY-013: 状態変更と対応付けの更新は互いを上書きせず、テナント境界と一意性を保つ。
func TestBindingMappingContract(t *testing.T) {
	testing_contract.RunBindingMapping(t, testing_contract.BindingMappingFixture{Repository: NewAgentWorkloadBindingRepository(), TenantA: "a", TenantB: "b", BundleID: "bundle", BindingID: "binding", OtherID: "other", AgentID: "agent", ReplacementAgentID: "replacement", Now: time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)})
}
