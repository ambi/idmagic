package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

func TestMcpResourceServerRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMcpResourceServerRepository()
	now := time.Now().UTC()

	t.Run("Save and FindByID", func(t *testing.T) {
		m := &domain.McpResourceServer{
			TenantID:         "tenant-1",
			ResourceServerID: "rs-1",
			Resource:         "https://mcp.example.com/tools/github",
			Name:             "GitHub MCP Tools",
			Scopes:           []string{"mcp.read"},
			State:            domain.McpResourceServerActive,
			CreatedAt:        now,
			UpdatedAt:        now,
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
			TenantID: "tenant-1", ResourceServerID: "rs-2",
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
		if found == nil || found.ResourceServerID != "rs-2" {
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

	t.Run("ListByTenant", func(t *testing.T) {
		list, err := repo.ListByTenant(ctx, "tenant-1")
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
