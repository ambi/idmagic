package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/idmanagement/group/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{Repository: NewGroupRepository(), TenantID: "tenant-contract", UserID: "user-contract", Group: &domain.Group{ID: "group-contract", TenantID: "tenant-contract", Name: "Contract Group", Roles: []string{}, CreatedAt: now, UpdatedAt: now}, Now: now}
	})
}
