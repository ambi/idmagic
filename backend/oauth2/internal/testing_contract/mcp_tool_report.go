package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// RunMcpToolReports は報告の置き換え、テナントの分離、削除を両保存先で確認する。
func RunMcpToolReports(t *testing.T, repo ports.McpToolReportRepository, tenantID, otherTenantID, serverID string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if found, err := repo.Find(ctx, tenantID, serverID); err != nil || found != nil {
		t.Fatalf("before any report: %+v %v", found, err)
	}
	first := &domain.McpToolReport{
		TenantID: tenantID, ResourceServerID: serverID, Complete: true, ReportedAt: now,
		Tools: []domain.ReportedTool{{Name: "read_document", Title: "閲覧", Description: "文書を読む", Arguments: []string{"document_id"}}, {Name: "delete_document"}},
	}
	if err := repo.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	second := &domain.McpToolReport{
		TenantID: tenantID, ResourceServerID: serverID, Complete: false, ReportedAt: now.Add(time.Minute),
		Tools: []domain.ReportedTool{{Name: "write_document"}},
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	found, err := repo.Find(ctx, tenantID, serverID)
	if err != nil || found == nil || found.Complete || len(found.Tools) != 1 || found.Tools[0].Name != "write_document" || !found.ReportedAt.Equal(second.ReportedAt) {
		t.Fatalf("replaced report=%+v err=%v", found, err)
	}
	if other, err := repo.Find(ctx, otherTenantID, serverID); err != nil || other != nil {
		t.Fatalf("another tenant read the report: %+v %v", other, err)
	}
	if err := repo.Delete(ctx, tenantID, serverID); err != nil {
		t.Fatal(err)
	}
	if found, err := repo.Find(ctx, tenantID, serverID); err != nil || found != nil {
		t.Fatalf("after delete: %+v %v", found, err)
	}
}
