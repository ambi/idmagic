package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
)

//spec:covers REQ-OAUTH2-085: メモリの条件付き保存は古い操作を復活させない。
func TestMcpCatalogCAS(t *testing.T) {
	testing_contract.RunMcpCatalogCAS(t, NewMcpResourceServerRepository(), "tenant-1", "catalog-cas", time.Now().UTC().Truncate(time.Microsecond))
}

//spec:covers REQ-OAUTH2-091: 上流設定と許可 Origin を、保存後の入力や読み取り値の書き換えで変更しない。
func TestManagedMcpConfigurationEditsRequireSave(t *testing.T) {
	repo := NewMcpResourceServerRepository()
	ctx := context.Background()
	server := &domain.McpResourceServer{TenantID: "tenant", ID: "managed", Resource: "https://idmagic.example/mcp", Hosting: &domain.HostedMcpConfiguration{UpstreamURL: "https://upstream.example/mcp", Authentication: "token_exchange", AllowedOrigins: []string{"https://agent.example"}}}
	if err := repo.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	server.Hosting.UpstreamURL = "https://changed.example"
	server.Hosting.AllowedOrigins[0] = "https://changed.example"
	loaded, err := repo.FindByID(ctx, "tenant", "managed")
	if err != nil || loaded.Hosting.UpstreamURL != "https://upstream.example/mcp" || loaded.Hosting.AllowedOrigins[0] != "https://agent.example" {
		t.Fatalf("input changed configuration: %+v %v", loaded, err)
	}
	loaded.Hosting.AllowedOrigins[0] = "https://changed.example"
	again, err := repo.FindByID(ctx, "tenant", "managed")
	if err != nil || again.Hosting.AllowedOrigins[0] != "https://agent.example" {
		t.Fatalf("read changed configuration: %+v %v", again, err)
	}
}

//spec:covers REQ-OAUTH2-064: 保存または読み取りで得たカタログの編集は、保存前の正本を変えない。
func TestToolCatalogEditsRequireSave(t *testing.T) {
	ctx := context.Background()
	repo := NewMcpResourceServerRepository()
	server := &domain.McpResourceServer{TenantID: "tenant-1", ID: "catalog", Resource: "https://catalog.example/", AccessRules: []domain.ResourceAccessRule{{Scopes: []string{"read"}, Tool: &domain.ResourceAccessTool{ID: "operation-" + "read_document", Name: "read_document", DisplayName: "閲覧"}}}}
	if err := repo.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	server.AccessRules[0].Tool.Name = "delete_document"
	server.AccessRules[0].Scopes[0] = "write"
	loaded, err := repo.FindByID(ctx, "tenant-1", "catalog")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccessRules[0].Tool.Name != "read_document" || loaded.AccessRules[0].Scopes[0] != "read" {
		t.Fatal("input changed saved catalog")
	}
	loaded.AccessRules[0].Tool.Name = "delete_document"
	again, err := repo.FindByResource(ctx, "tenant-1", "https://catalog.example/")
	if err != nil {
		t.Fatal(err)
	}
	if again.AccessRules[0].Tool.Name != "read_document" {
		t.Fatal("read changed saved catalog")
	}
}

func TestMcpResourceServerRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMcpResourceServerRepository()
	now := time.Now().UTC()

	t.Run("Save and FindByID", func(t *testing.T) {
		m := &domain.McpResourceServer{
			TenantID:  "tenant-1",
			ID:        "rs-1",
			Resource:  "https://mcp.example.com/tools/github",
			Name:      "GitHub MCP Tools",
			Scopes:    []string{"mcp.read"},
			State:     domain.McpResourceServerActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := repo.Save(ctx, m); err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-1", "rs-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected resource server to be found")
		}
		if found.Resource != "https://mcp.example.com/tools/github" {
			t.Errorf("unexpected resource: %q", found.Resource)
		}

		notFound, err := repo.FindByID(ctx, "tenant-1", "non-existent")
		if err != nil {
			t.Fatal(err)
		}
		if notFound != nil {
			t.Error("expected nil for non-existing resource_server_id")
		}
	})

	t.Run("FindByResource is tenant scoped", func(t *testing.T) {
		m := &domain.McpResourceServer{
			TenantID: "tenant-1", ID: "rs-2",
			Resource: "https://mcp.example.com/tools/slack", Name: "Slack MCP",
			Scopes: []string{"mcp.read"}, State: domain.McpResourceServerActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Save(ctx, m); err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByResource(ctx, "tenant-1", "https://mcp.example.com/tools/slack")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil || found.ID != "rs-2" {
			t.Fatal("expected to find rs-2 by resource within tenant-1")
		}

		crossTenant, err := repo.FindByResource(ctx, "tenant-other", "https://mcp.example.com/tools/slack")
		if err != nil {
			t.Fatal(err)
		}
		if crossTenant != nil {
			t.Error("expected resource lookup to be tenant scoped")
		}
	})

	t.Run("ListAll", func(t *testing.T) {
		list, err := repo.ListAll(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 resource servers for tenant-1, got %d", len(list))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := repo.Delete(ctx, "tenant-1", "rs-1"); err != nil {
			t.Fatal(err)
		}
		found, err := repo.FindByID(ctx, "tenant-1", "rs-1")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected rs-1 to be deleted")
		}
	})
}
