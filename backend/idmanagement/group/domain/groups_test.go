package domain_test

import (
	"slices"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

func TestEffectiveRolesUnionSortedDedup(t *testing.T) {
	groups := []*groupdomain.Group{
		{Roles: []string{"catalog:read", "invoice:read"}},
		{Roles: []string{"catalog:read", "support:read"}},
		nil,
	}
	got := groupdomain.EffectiveRoles([]string{"admin", "support:read"}, groups)
	want := []string{"admin", "catalog:read", "invoice:read", "support:read"}
	if !slices.Equal(got, want) {
		t.Fatalf("EffectiveRoles = %v, want %v", got, want)
	}
}

func TestEffectiveRolesEmptyGroupsEqualsUserRoles(t *testing.T) {
	got := groupdomain.EffectiveRoles([]string{"admin", "auditor"}, nil)
	want := []string{"admin", "auditor"}
	if !slices.Equal(got, want) {
		t.Fatalf("EffectiveRoles = %v, want %v", got, want)
	}
}

func TestGroupValidate(t *testing.T) {
	now := time.Now().UTC()
	valid := groupdomain.Group{ID: "group_x", TenantID: tenancydomain.DefaultTenantID, Name: "engineering", Roles: []string{"catalog:read"}, CreatedAt: now, UpdatedAt: now}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid group rejected: %v", err)
	}
	missingName := groupdomain.Group{ID: "group_x", TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now}
	if err := missingName.Validate(); err == nil {
		t.Fatal("group without name was accepted")
	}
}

func TestNewGroupIDIsUUID(t *testing.T) {
	id, err := groupdomain.NewGroupID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 36 {
		t.Fatalf("unexpected group id %q", id)
	}
}
