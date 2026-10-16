// Package testing_contract defines the shared OAuth 2.0 persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

type Fixture struct {
	DetailTypes ports.AuthorizationDetailTypeRepository
	TenantA     string
	TenantB     string
	Now         time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	detailType := &domain.AuthorizationDetailType{
		TenantID: f.TenantA, Type: "payment_initiation", Description: "Payment initiation details",
		DisplayTemplate: "Pay {{.amount}}", State: domain.DetailTypeEnabled,
		Schema: domain.AuthorizationDetailsSchema{Rules: []domain.AuthorizationDetailFieldRule{
			{Name: "currency", Semantics: domain.DetailFieldEnum, Allowed: []string{"JPY", "USD"}},
		}},
		CreatedAt: f.Now, UpdatedAt: f.Now,
	}
	if err := f.DetailTypes.Save(ctx, detailType); err != nil {
		t.Fatalf("Save: %v", err)
	}
	found, err := f.DetailTypes.FindByType(ctx, f.TenantA, detailType.Type)
	if err != nil || found == nil || found.DisplayTemplate != detailType.DisplayTemplate || len(found.Schema.Rules) != 1 {
		t.Fatalf("FindByType = (%+v, %v)", found, err)
	}
	if leaked, err := f.DetailTypes.FindByType(ctx, f.TenantB, detailType.Type); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByType = (%+v, %v)", leaked, err)
	}
	listed, err := f.DetailTypes.ListAll(ctx, f.TenantA)
	if err != nil || len(listed) != 1 || listed[0].Type != detailType.Type {
		t.Fatalf("ListAll = (%+v, %v)", listed, err)
	}
	found.Schema.Rules[0].Allowed[0] = "mutated"
	again, err := f.DetailTypes.FindByType(ctx, f.TenantA, detailType.Type)
	if err != nil || again == nil || again.Schema.Rules[0].Allowed[0] == "mutated" {
		t.Fatalf("stored value aliased caller memory: (%+v, %v)", again, err)
	}
	if err := f.DetailTypes.Delete(ctx, f.TenantA, detailType.Type); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted, err := f.DetailTypes.FindByType(ctx, f.TenantA, detailType.Type); err != nil || deleted != nil {
		t.Fatalf("FindByType after Delete = (%+v, %v)", deleted, err)
	}
}
