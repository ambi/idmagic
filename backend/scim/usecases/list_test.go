package usecases_test

import (
	"context"
	"testing"

	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/adapters/persistence/memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	scimmemory "github.com/ambi/idmagic/backend/scim/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/scim/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// ListUsers/ListGroups は tenant_id で厳密に境界を切り、他 tenant の resource を
// 一覧にも filter match にも漏らさない (wi-238 Risk Notes、scenario
// "SCIM clientはUsersとGroups collectionを検索できる")。
func TestListUsersTenantIsolation(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()
	u := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})

	const tenantB = "tenant-b"
	if _, err := u.CreateUser(ctx, tenancydomain.DefaultTenantID, map[string]any{"userName": "a-user@example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := u.CreateUser(ctx, tenantB, map[string]any{"userName": "b-user@example.com"}); err != nil {
		t.Fatal(err)
	}

	result, err := u.ListUsers(ctx, tenancydomain.DefaultTenantID, usecases.ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1 (tenant-b user must not leak)", result.Total)
	}
	if got := result.Items[0]["userName"].(string); got != "a-user@example.com" {
		t.Errorf("userName = %q, want a-user@example.com", got)
	}

	// tenant-b の token で他 tenant の userName を filter しても一致しない。
	result, err = u.ListUsers(ctx, tenantB, usecases.ListQuery{Filter: `userName eq "a-user@example.com"`})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 {
		t.Fatalf("Total = %d, want 0 (cross-tenant filter must not match)", result.Total)
	}
}

func TestListGroupsTenantIsolation(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()
	u := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})

	const tenantB = "tenant-b"
	if _, err := u.CreateGroup(ctx, tenancydomain.DefaultTenantID, map[string]any{"displayName": "A-Team"}); err != nil {
		t.Fatal(err)
	}
	if _, err := u.CreateGroup(ctx, tenantB, map[string]any{"displayName": "B-Team"}); err != nil {
		t.Fatal(err)
	}

	result, err := u.ListGroups(ctx, tenancydomain.DefaultTenantID, usecases.ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1 (tenant-b group must not leak)", result.Total)
	}
	if got := result.Items[0]["displayName"].(string); got != "A-Team" {
		t.Errorf("displayName = %q, want A-Team", got)
	}
}
