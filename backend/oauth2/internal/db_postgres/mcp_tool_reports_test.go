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

//spec:covers REQ-OAUTH2-086: 指紋の列を足す前に保存した行は内容を比べられないので、次の報告で変更前の一覧を作らない。
func TestMcpToolReportWithoutAFingerprintDoesNotBecomeAPreviousList(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db).ID
	now := testClock()
	id := newUUID(t)
	if err := (&oauthpg.McpResourceServerRepository{Pool: db}).Save(ctx, &domain.McpResourceServer{TenantID: tenantID, ID: id, Resource: "https://" + id + ".example/", Name: "Legacy", Scopes: []string{"mcp"}, State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO mcp_tool_reports (tenant_id, resource_server_id, tools, complete, reported_at) VALUES ($1, $2, '[{"name":"old"}]', true, $3)`, tenantID, id, now); err != nil {
		t.Fatal(err)
	}
	reports := &oauthpg.McpToolReportRepository{Pool: db}
	if err := reports.Save(ctx, &domain.McpToolReport{TenantID: tenantID, ResourceServerID: id, Complete: true, ReportedAt: now.Add(time.Minute), Tools: []domain.ReportedTool{{Name: "new"}}}); err != nil {
		t.Fatal(err)
	}
	found, err := reports.Find(ctx, tenantID, id)
	if err != nil || found == nil || found.Tools[0].Name != "new" || found.Previous != nil || !found.ChangedAt.IsZero() {
		t.Fatalf("report=%+v err=%v, want the new list without a previous one", found, err)
	}
}
