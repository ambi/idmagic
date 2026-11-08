package db_postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthpg "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-OAUTH2-086, REQ-OAUTH2-087: PostgreSQLのカタログは置き換えとテナントの分離を守り、登録の削除で消える。
func TestMcpToolCatalogRepository(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenantID, otherTenantID := seedTenant(t, db).ID, seedTenant(t, db).ID
	servers := &oauthpg.McpResourceServerRepository{Pool: db}
	catalogs := &oauthpg.McpToolCatalogRepository{Pool: db}
	now := testClock()
	seedServer := func() string {
		t.Helper()
		id := newUUID(t)
		if err := servers.Save(ctx, &domain.McpResourceServer{TenantID: tenantID, ID: id, Resource: "https://" + id + ".example/", Name: "Reports", Scopes: []string{"mcp"}, State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	testing_contract.RunMcpToolCatalogs(t, catalogs, tenantID, otherTenantID, seedServer(), now)

	cascaded := seedServer()
	if err := catalogs.Save(ctx, &domain.McpToolCatalog{TenantID: tenantID, ResourceServerID: cascaded, Complete: true, ReceivedAt: now, Tools: []domain.CatalogTool{{Name: "read_document"}}}); err != nil {
		t.Fatal(err)
	}
	if err := servers.Delete(ctx, tenantID, cascaded); err != nil {
		t.Fatal(err)
	}
	if found, err := catalogs.Find(ctx, tenantID, cascaded); err != nil || found != nil {
		t.Fatalf("report survived its server: %+v %v", found, err)
	}
}

//spec:covers REQ-OAUTH2-086: 指紋の列を足す前に保存した行は内容を比べられないので、次のカタログで変更前の一覧を作らない。
func TestMcpToolCatalogWithoutAFingerprintDoesNotBecomeAPreviousList(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db).ID
	now := testClock()
	id := newUUID(t)
	if err := (&oauthpg.McpResourceServerRepository{Pool: db}).Save(ctx, &domain.McpResourceServer{TenantID: tenantID, ID: id, Resource: "https://" + id + ".example/", Name: "Legacy", Scopes: []string{"mcp"}, State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO mcp_tool_catalogs (tenant_id, resource_server_id, tools, complete, received_at) VALUES ($1, $2, '[{"name":"old"}]', true, $3)`, tenantID, id, now); err != nil {
		t.Fatal(err)
	}
	catalogs := &oauthpg.McpToolCatalogRepository{Pool: db}
	if err := catalogs.Save(ctx, &domain.McpToolCatalog{TenantID: tenantID, ResourceServerID: id, Complete: true, ReceivedAt: now.Add(time.Minute), Tools: []domain.CatalogTool{{Name: "new"}}}); err != nil {
		t.Fatal(err)
	}
	found, err := catalogs.Find(ctx, tenantID, id)
	if err != nil || found == nil || found.Tools[0].Name != "new" || found.Previous != nil || !found.ChangedAt.IsZero() {
		t.Fatalf("report=%+v err=%v, want the new list without a previous one", found, err)
	}
}
