package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// RunMcpCatalogCAS は操作の削除と再登録を挟んだ古い保存の拒否を両保存先で確認する。
func RunMcpCatalogCAS(t *testing.T, repo ports.McpResourceServerRepository, tenantID, id string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	rule := domain.ResourceAccessRule{ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"read"}, Tool: &domain.ResourceAccessTool{ID: "operation-first", Name: "read", DisplayName: "閲覧"}}
	server := &domain.McpResourceServer{TenantID: tenantID, ID: id, Resource: "https://" + id + ".example/", Name: "Catalog", Scopes: []string{"read"}, AccessRules: []domain.ResourceAccessRule{rule}, State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	stale, err := repo.FindByID(ctx, tenantID, id)
	if err != nil {
		t.Fatal(err)
	}
	previous := server.UpdatedAt
	server.AccessRules = nil
	server.UpdatedAt = now.Add(time.Microsecond)
	if saved, err := repo.SaveIfUnchanged(ctx, server, previous); err != nil || !saved {
		t.Fatalf("remove: %v %v", saved, err)
	}
	previous = server.UpdatedAt
	rule.Tool.ID = "operation-second"
	server.AccessRules = []domain.ResourceAccessRule{rule}
	server.UpdatedAt = now.Add(2 * time.Microsecond)
	if saved, err := repo.SaveIfUnchanged(ctx, server, previous); err != nil || !saved {
		t.Fatalf("recreate: %v %v", saved, err)
	}
	stale.Name = "stale"
	stale.UpdatedAt = now.Add(time.Second)
	if saved, err := repo.SaveIfUnchanged(ctx, stale, now); err != nil || saved {
		t.Fatalf("stale update: %v %v", saved, err)
	}
	current, err := repo.FindByID(ctx, tenantID, id)
	if err != nil || current.Name != "Catalog" || current.AccessRules[0].Tool.ID != "operation-second" {
		t.Fatalf("current=%+v err=%v", current, err)
	}
	if err := repo.Delete(ctx, tenantID, id); err != nil {
		t.Fatal(err)
	}
	if saved, err := repo.SaveIfUnchanged(ctx, server, server.UpdatedAt); err != nil || saved {
		t.Fatalf("deleted: %v %v", saved, err)
	}
}
