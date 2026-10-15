// Package testing_contract defines the shared tenancy persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

type Fixture struct {
	Repository ports.TenantRepository
	Tenant     *domain.Tenant
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.Repository.Save(ctx, f.Tenant); err != nil {
		t.Fatalf("Save: %v", err)
	}
	byID, err := f.Repository.FindByID(ctx, f.Tenant.ID)
	if err != nil || byID == nil || byID.Realm != f.Tenant.Realm || byID.DisplayName != f.Tenant.DisplayName {
		t.Fatalf("FindByID = (%+v, %v)", byID, err)
	}
	byRealm, err := f.Repository.FindByRealm(ctx, f.Tenant.Realm)
	if err != nil || byRealm == nil || byRealm.ID != f.Tenant.ID {
		t.Fatalf("FindByRealm = (%+v, %v)", byRealm, err)
	}
	updated := *f.Tenant
	updated.DisplayName = f.Tenant.DisplayName + " updated"
	updated.UpdatedAt = updated.UpdatedAt.Add(time.Minute)
	if err := f.Repository.Save(ctx, &updated); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	got, err := f.Repository.FindByID(ctx, f.Tenant.ID)
	if err != nil || got == nil || got.DisplayName != updated.DisplayName {
		t.Fatalf("FindByID after update = (%+v, %v)", got, err)
	}
	all, err := f.Repository.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	found := false
	for _, tenant := range all {
		if tenant.ID == f.Tenant.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("FindAll omitted tenant %q: %+v", f.Tenant.ID, all)
	}
	missing, err := f.Repository.FindByID(ctx, "00000000-0000-0000-0000-0000000000ff")
	if err != nil || missing != nil {
		t.Fatalf("FindByID missing = (%+v, %v)", missing, err)
	}
}
