package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestAuthorizationDetailTypeRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewAuthorizationDetailTypeRepository()

	t.Run("Save and FindByType", func(t *testing.T) {
		detailType := &spec.AuthorizationDetailType{
			TenantID:    "tenant-1",
			Type:        "payment",
			Description: "Payment details type",
			Schema: spec.AuthorizationDetailsSchema{
				Rules: []spec.AuthorizationDetailFieldRule{
					{
						Allowed: []string{"USD", "JPY"},
					},
				},
			},
			State:     spec.DetailTypeEnabled,
			CreatedAt: time.Now(),
		}

		err := repo.Save(ctx, detailType)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByType(ctx, "tenant-1", "payment")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected detail type to be found")
		}
		if found.Type != "payment" {
			t.Errorf("expected Type to be 'payment', got %q", found.Type)
		}
		if len(found.Schema.Rules) != 1 || len(found.Schema.Rules[0].Allowed) != 2 {
			t.Errorf("unexpected schema rules: %v", found.Schema.Rules)
		}

		// 存在しないタイプ
		notfound, err := repo.FindByType(ctx, "tenant-1", "non-existent")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing type")
		}
	})

	t.Run("Seed", func(t *testing.T) {
		detailType := &spec.AuthorizationDetailType{
			TenantID: "tenant-1",
			Type:     "seed-type",
			State:    spec.DetailTypeEnabled,
		}
		//nolint:contextcheck // memory repo Seed doesn't take context
		repo.Seed(detailType)

		found, err := repo.FindByType(ctx, "tenant-1", "seed-type")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected seeded type to be found")
		}
	})

	t.Run("ListByTenant", func(t *testing.T) {
		// すでに payment と seed-type が tenant-1 に存在する
		// さらに追加してソート順を確認する
		dtC := &spec.AuthorizationDetailType{
			TenantID: "tenant-1",
			Type:     "c-type",
			State:    spec.DetailTypeEnabled,
		}
		dtB := &spec.AuthorizationDetailType{
			TenantID: "tenant-1",
			Type:     "b-type",
			State:    spec.DetailTypeEnabled,
		}
		dtOther := &spec.AuthorizationDetailType{
			TenantID: "tenant-other",
			Type:     "other-type",
			State:    spec.DetailTypeEnabled,
		}

		_ = repo.Save(ctx, dtC)
		_ = repo.Save(ctx, dtB)
		_ = repo.Save(ctx, dtOther)

		list, err := repo.ListByTenant(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		// tenant-1 には payment, seed-type, c-type, b-type の 4 つがあるはず
		if len(list) != 4 {
			t.Fatalf("expected 4 detail types, got %d", len(list))
		}
		// Type 順（b-type, c-type, payment, seed-type）でソートされていることを検証
		if list[0].Type != "b-type" || list[1].Type != "c-type" || list[2].Type != "payment" || list[3].Type != "seed-type" {
			t.Errorf("list is not sorted by Type: %s, %s, %s, %s", list[0].Type, list[1].Type, list[2].Type, list[3].Type)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "tenant-1", "payment")
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByType(ctx, "tenant-1", "payment")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected payment to be deleted")
		}
	})
}
