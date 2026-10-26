package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/idmanagement/internal/group/testing_contract"
)

//spec:covers REQ-IDMANAGEMENT-092, REQ-IDMANAGEMENT-093: メモリ保存でも所属を保って未承認ロールを除き、通常保存で承認の取消しを戻させない。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{Repository: NewGroupRepository(), TenantID: "tenant-contract", UserID: "user-contract", Group: &domain.Group{ID: "group-contract", TenantID: "tenant-contract", Name: "Contract Group", Roles: []string{}, CreatedAt: now, UpdatedAt: now}, Now: now}
	})
}
