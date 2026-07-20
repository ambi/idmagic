package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestTenantRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewTenantRepository()

	t.Run("Save and FindByID", func(t *testing.T) {
		tenant := &domain.Tenant{
			ID:          "tenant-1",
			DisplayName: "Tenant One",
			Status:      "Active",
			CreatedAt:   time.Now(),
		}

		err := repo.Save(ctx, tenant)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected tenant to be found")
		}
		if found.DisplayName != "Tenant One" {
			t.Errorf("expected DisplayName to be 'Tenant One', got %q", found.DisplayName)
		}

		// 存在しないテナント
		notfound, err := repo.FindByID(ctx, "tenant-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing tenant")
		}
	})

	t.Run("FindAll", func(t *testing.T) {
		// すでに tenant-1 が存在する
		t3 := &domain.Tenant{ID: "tenant-3", DisplayName: "Tenant Three"}
		t2 := &domain.Tenant{ID: "tenant-2", DisplayName: "Tenant Two"}

		_ = repo.Save(ctx, t3)
		_ = repo.Save(ctx, t2)

		list, err := repo.FindAll(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// 3 件あるはず
		if len(list) != 3 {
			t.Fatalf("expected 3 tenants, got %d", len(list))
		}
		// ID 順（tenant-1, tenant-2, tenant-3）でソートされていることを検証
		if list[0].ID != "tenant-1" || list[1].ID != "tenant-2" || list[2].ID != "tenant-3" {
			t.Errorf("list is not sorted by ID: %s, %s, %s", list[0].ID, list[1].ID, list[2].ID)
		}
	})
}
