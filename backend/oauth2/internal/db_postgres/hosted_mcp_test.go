package db_postgres_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthpg "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-OAUTH2-091: PostgreSQL でも上流設定を保存して各照会から返し、条件付き更新とテナント境界を保つ。
func TestManagedMcpConfigurationPersistsInPostgres(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenant, other := seedTenant(t, db).ID, seedTenant(t, db).ID
	repo := &oauthpg.McpResourceServerRepository{Pool: db}
	now := testClock()
	server := &domain.McpResourceServer{TenantID: tenant, ID: newUUID(t), Resource: "https://managed.example/mcp", Name: "Managed", Scopes: []string{"mcp"}, ClientID: "evaluator", Hosting: &domain.HostedMcpConfiguration{UpstreamURL: "https://upstream.example/mcp", Authentication: "token_vault", ConnectorID: "external", AllowedOrigins: []string{"https://agent.example"}}, State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	for name, load := range map[string]func() (*domain.McpResourceServer, error){
		"id":       func() (*domain.McpResourceServer, error) { return repo.FindByID(ctx, tenant, server.ID) },
		"resource": func() (*domain.McpResourceServer, error) { return repo.FindByResource(ctx, tenant, server.Resource) },
		"client":   func() (*domain.McpResourceServer, error) { return repo.FindByClientID(ctx, tenant, server.ClientID) },
	} {
		loaded, err := load()
		if err != nil || loaded == nil || !reflect.DeepEqual(loaded.Hosting, server.Hosting) {
			t.Fatalf("%s lost configuration: %+v %v", name, loaded, err)
		}
	}
	list, err := repo.ListAll(ctx, tenant)
	if err != nil || len(list) != 1 || !reflect.DeepEqual(list[0].Hosting, server.Hosting) {
		t.Fatalf("list lost configuration: %+v %v", list, err)
	}
	if loaded, err := repo.FindByID(ctx, other, server.ID); err != nil || loaded != nil {
		t.Fatalf("cross-tenant configuration: %+v %v", loaded, err)
	}
	previous := server.UpdatedAt
	server.Hosting.UpstreamURL = "https://changed.example/mcp"
	server.UpdatedAt = previous.Add(time.Second)
	if saved, err := repo.SaveIfUnchanged(ctx, server, previous); err != nil || !saved {
		t.Fatalf("update: %v %v", saved, err)
	}
	if saved, err := repo.SaveIfUnchanged(ctx, server, previous); err != nil || saved {
		t.Fatalf("stale update: %v %v", saved, err)
	}
	loaded, err := repo.FindByID(ctx, tenant, server.ID)
	if err != nil || loaded.Hosting.UpstreamURL != server.Hosting.UpstreamURL {
		t.Fatalf("update lost configuration: %+v %v", loaded, err)
	}
}
