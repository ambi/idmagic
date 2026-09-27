// Package testing_contract defines the shared SCIM reference repository contract.
package testing_contract

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

type Fixture struct {
	Repository ports.ScimRepository
	TenantID   string
	UserID     string
	GroupID    string
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	userRef := &ports.ScimUserRef{TenantID: f.TenantID, ScimID: "scim-user-contract", UserID: f.UserID}
	if err := f.Repository.SaveUserRef(ctx, userRef); err != nil {
		t.Fatalf("SaveUserRef: %v", err)
	}
	if got, err := f.Repository.FindUserRefByScimID(ctx, f.TenantID, userRef.ScimID); err != nil || got == nil || got.UserID != f.UserID {
		t.Fatalf("FindUserRefByScimID = (%+v, %v)", got, err)
	}
	if got, err := f.Repository.FindUserRefByUserID(ctx, f.TenantID, f.UserID); err != nil || got == nil || got.ScimID != userRef.ScimID {
		t.Fatalf("FindUserRefByUserID = (%+v, %v)", got, err)
	}
	groupRef := &ports.ScimGroupRef{TenantID: f.TenantID, ScimID: "scim-group-contract", GroupID: f.GroupID}
	if err := f.Repository.SaveGroupRef(ctx, groupRef); err != nil {
		t.Fatalf("SaveGroupRef: %v", err)
	}
	if got, err := f.Repository.FindGroupRefByGroupID(ctx, f.TenantID, f.GroupID); err != nil || got == nil || got.ScimID != groupRef.ScimID {
		t.Fatalf("FindGroupRefByGroupID = (%+v, %v)", got, err)
	}
	if err := f.Repository.DeleteUserRef(ctx, f.TenantID, userRef.ScimID); err != nil {
		t.Fatalf("DeleteUserRef: %v", err)
	}
	if err := f.Repository.DeleteGroupRef(ctx, f.TenantID, groupRef.ScimID); err != nil {
		t.Fatalf("DeleteGroupRef: %v", err)
	}
}
