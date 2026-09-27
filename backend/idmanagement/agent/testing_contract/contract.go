// Package testing_contract defines the shared agent persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	"github.com/ambi/idmagic/backend/idmanagement/agent/ports"
)

type Fixture struct {
	Repository ports.AgentRepository
	TenantID   string
	Agent      *domain.Agent
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.Repository.Save(ctx, f.Agent); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.FindByID(ctx, f.TenantID, f.Agent.ID)
	if err != nil || got == nil || got.Name != f.Agent.Name {
		t.Fatalf("FindByID = (%+v, %v)", got, err)
	}
	byClient, err := f.Repository.FindByClientID(ctx, f.TenantID, "00000000-0000-0000-0000-0000000000ff")
	if err != nil || byClient != nil {
		t.Fatalf("FindByClientID missing = (%+v, %v)", byClient, err)
	}
	all, err := f.Repository.ListAll(ctx, f.TenantID)
	if err != nil || len(all) != 1 || all[0].ID != f.Agent.ID {
		t.Fatalf("ListAll = (%+v, %v)", all, err)
	}
	updated := *f.Agent
	updated.Name = "updated-agent"
	updated.UpdatedAt = updated.UpdatedAt.Add(time.Minute)
	if err := f.Repository.Save(ctx, &updated); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	if err := f.Repository.Delete(ctx, f.TenantID, f.Agent.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.FindByID(ctx, f.TenantID, f.Agent.ID); err != nil || got != nil {
		t.Fatalf("FindByID after delete = (%+v, %v)", got, err)
	}
}
