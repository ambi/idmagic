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

	// Update は呼び出し側が先に読んだ値ではなく、保存先の最新の値に変更を当てる。
	described := "described by Update"
	afterUpdate, err := f.Repository.Update(ctx, f.TenantID, f.Agent.ID, func(latest *domain.Agent) (bool, error) {
		if latest.Name != "updated-agent" {
			t.Errorf("Update received name %q, want the latest saved name", latest.Name)
		}
		latest.Description = &described
		return true, nil
	})
	if err != nil || afterUpdate == nil {
		t.Fatalf("Update = (%+v, %v)", afterUpdate, err)
	}
	if got, err := f.Repository.FindByID(ctx, f.TenantID, f.Agent.ID); err != nil || got == nil ||
		got.Name != "updated-agent" || got.Description == nil || *got.Description != described {
		t.Fatalf("FindByID after Update = (%+v, %v)", got, err)
	}
	if _, err := f.Repository.Update(ctx, f.TenantID, f.Agent.ID, func(latest *domain.Agent) (bool, error) {
		latest.Name = "discarded"
		return false, nil
	}); err != nil {
		t.Fatalf("Update without change: %v", err)
	}
	if got, err := f.Repository.FindByID(ctx, f.TenantID, f.Agent.ID); err != nil || got == nil || got.Name != "updated-agent" {
		t.Fatalf("Update without change saved: (%+v, %v)", got, err)
	}
	called := false
	if missing, err := f.Repository.Update(ctx, f.TenantID, "00000000-0000-0000-0000-0000000000fe", func(*domain.Agent) (bool, error) {
		called = true
		return true, nil
	}); err != nil || missing != nil || called {
		t.Fatalf("Update of a missing agent = (%+v, %v, called=%v)", missing, err, called)
	}

	if err := f.Repository.Delete(ctx, f.TenantID, f.Agent.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.FindByID(ctx, f.TenantID, f.Agent.ID); err != nil || got != nil {
		t.Fatalf("FindByID after delete = (%+v, %v)", got, err)
	}
}
