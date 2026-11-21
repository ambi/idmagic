package db_memory_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/idgovernance/internal/db_memory"
	"github.com/ambi/idmagic/backend/idgovernance/internal/testing_contract"
)

//spec:covers REQ-IDGOVERNANCE-021, REQ-IDGOVERNANCE-022, REQ-IDGOVERNANCE-025: メモリでも各回の開始と回答を直列化し、他テナントの操作を拒否する。
func TestAccessReviewPersistenceContract(t *testing.T) {
	r := db_memory.NewAccessReviewRepository()
	testing_contract.RunAccessReviews(t, testing_contract.AccessReviewFixture{Plans: r, Reviews: r, TenantA: "tenant-a", TenantB: "tenant-b"})
}
