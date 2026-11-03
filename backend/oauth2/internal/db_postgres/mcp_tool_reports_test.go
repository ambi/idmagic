package db_postgres_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthpg "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-OAUTH2-086, REQ-OAUTH2-087: PostgreSQLの報告は置き換えとテナントの分離を守り、登録の削除で消える。
func TestMcpToolReportRepository(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenantID, otherTenantID := seedTenant(t, db).ID, seedTenant(t, db).ID
	servers := &oauthpg.McpResourceServerRepository{Pool: db}
	reports := &oauthpg.McpToolReportRepository{Pool: db}
	now := testClock()
	seedServer := func() string {
		t.Helper()
		id := newUUID(t)
		if err := servers.Save(ctx, &domain.McpResourceServer{TenantID: tenantID, ID: id, Resource: "https://" + id + ".example/", Name: "Reports", Scopes: []string{"mcp"}, State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	testing_contract.RunMcpToolReports(t, reports, tenantID, otherTenantID, seedServer(), now)

	cascaded := seedServer()
	if err := reports.Save(ctx, &domain.McpToolReport{TenantID: tenantID, ResourceServerID: cascaded, Complete: true, ReportedAt: now, Tools: []domain.ReportedTool{{Name: "read_document"}}}); err != nil {
		t.Fatal(err)
	}
	if err := servers.Delete(ctx, tenantID, cascaded); err != nil {
		t.Fatal(err)
	}
	if found, err := reports.Find(ctx, tenantID, cascaded); err != nil || found != nil {
		t.Fatalf("report survived its server: %+v %v", found, err)
	}
}
