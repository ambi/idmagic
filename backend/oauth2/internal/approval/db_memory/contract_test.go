package db_memory_test

import (
	"context"
	"testing"

	approvaldb "github.com/ambi/idmagic/backend/oauth2/internal/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/approval/testing_contract"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-OAUTH2-093: 同じ操作の保留の再利用、一人 10 件の上限、並行の起票の直列化、承認後の一回の使用
func TestResourceServerStartContract(t *testing.T) {
	testing_contract.RunResourceServerStartContract(t, func(t *testing.T) testing_contract.ResourceServerStartFixture {
		t.Helper()
		return testing_contract.ResourceServerStartFixture{
			Store:   approvaldb.NewApprovalRequestStore(),
			Context: tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-contract"}, "", ""),
			UserID:  "alice", ClientID: "client-contract", OtherUser: "bob",
			NewID: func() string {
				id, err := approvaldomain.NewApprovalRequestID()
				if err != nil {
					t.Fatal(err)
				}
				return id
			},
		}
	})
}
