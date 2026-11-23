package db_memory_test

import (
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
)

//spec:covers REQ-OAUTH2-097: 影響度の保存、解除、テナント分離と複製の共有契約をメモリで満たす。
func TestPermissionImpactPersistence(t *testing.T) {
	testing_contract.RunPermissionImpact(t, testing_contract.ImpactFixture{Types: memory.NewAuthorizationDetailTypeRepository(), Servers: memory.NewMcpResourceServerRepository(), TenantID: "tenant-a", OtherTenantID: "tenant-b", ServerID: "server", Now: time.Now().UTC().Truncate(time.Microsecond)})
}
