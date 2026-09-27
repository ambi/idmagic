// Package testing_contract defines the shared application persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
)

type Fixture struct {
	Categories ports.ApplicationCategoryRepository
	TenantA    string
	TenantB    string
	CategoryA  string
	CategoryB  string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	categories := []*domain.ApplicationCategory{
		{TenantID: f.TenantA, ID: f.CategoryA, Name: "Beta", Position: 2, CreatedAt: f.Now, UpdatedAt: f.Now},
		{TenantID: f.TenantA, ID: f.CategoryB, Name: "Alpha", Position: 1, CreatedAt: f.Now, UpdatedAt: f.Now},
	}
	for _, category := range categories {
		if err := f.Categories.Save(ctx, category); err != nil {
			t.Fatalf("Save(%q): %v", category.ID, err)
		}
	}
	listed, err := f.Categories.ListAll(ctx, f.TenantA)
	if err != nil || len(listed) != 2 || listed[0].ID != f.CategoryB || listed[1].ID != f.CategoryA {
		t.Fatalf("ListAll = (%+v, %v)", listed, err)
	}
	found, err := f.Categories.FindByID(ctx, f.TenantA, f.CategoryA)
	if err != nil || found == nil || found.Name != "Beta" {
		t.Fatalf("FindByID = (%+v, %v)", found, err)
	}
	if leaked, err := f.Categories.FindByID(ctx, f.TenantB, f.CategoryA); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByID = (%+v, %v)", leaked, err)
	}
	found.Name = "mutated"
	again, err := f.Categories.FindByID(ctx, f.TenantA, f.CategoryA)
	if err != nil || again == nil || again.Name != "Beta" {
		t.Fatalf("stored category aliased caller memory: (%+v, %v)", again, err)
	}
	if err := f.Categories.Delete(ctx, f.TenantA, f.CategoryA); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted, err := f.Categories.FindByID(ctx, f.TenantA, f.CategoryA); err != nil || deleted != nil {
		t.Fatalf("FindByID after Delete = (%+v, %v)", deleted, err)
	}
}
