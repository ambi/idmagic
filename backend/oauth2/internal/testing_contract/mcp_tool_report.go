package testing_contract

import (
	"context"
	"reflect"
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
	defer func() {
		if t.Failed() {
			return
		}
		// 最初の報告には比べる相手がないので、変更前の一覧を作らない。
		fresh := &domain.McpToolReport{TenantID: tenantID, ResourceServerID: serverID, Complete: true, ReportedAt: now, Tools: []domain.ReportedTool{{Name: "a"}}}
		if err := repo.Save(ctx, fresh); err != nil {
			t.Fatal(err)
		}
		if found, err := repo.Find(ctx, tenantID, serverID); err != nil || found == nil || found.Previous != nil || !found.ChangedAt.IsZero() {
			t.Fatalf("the first report has a previous list: %+v %v", found, err)
		}
		if err := repo.Delete(ctx, tenantID, serverID); err != nil {
			t.Fatal(err)
		}
	}()
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
	// 内容の変わった報告は、直前の一覧と complete を残し、受信の時刻を変更の時刻にする (REQ-OAUTH2-086)。
	if found.Previous == nil || !reflect.DeepEqual(*found.Previous, domain.ReportedToolList{Tools: first.Tools, Complete: true}) || !found.ChangedAt.Equal(second.ReportedAt) {
		t.Fatalf("a changed report did not keep the previous list: previous=%+v changed_at=%v", found.Previous, found.ChangedAt)
	}
	// 同じ内容の報告は、受信の時刻と complete だけを変え、変更前の一覧を保つ。入力の Previous は読まない。
	same := &domain.McpToolReport{
		TenantID: tenantID, ResourceServerID: serverID, Complete: true, ReportedAt: now.Add(2 * time.Minute),
		Tools:    []domain.ReportedTool{{Name: "write_document"}},
		Previous: &domain.ReportedToolList{Tools: []domain.ReportedTool{{Name: "forged"}}}, ChangedAt: now.Add(time.Hour),
	}
	if err := repo.Save(ctx, same); err != nil {
		t.Fatal(err)
	}
	found, err = repo.Find(ctx, tenantID, serverID)
	if err != nil || found == nil || !found.Complete || !found.ReportedAt.Equal(same.ReportedAt) {
		t.Fatalf("the same report was not saved: %+v %v", found, err)
	}
	if found.Previous == nil || !reflect.DeepEqual(found.Previous.Tools, first.Tools) || !found.ChangedAt.Equal(second.ReportedAt) {
		t.Fatalf("the same report replaced the previous list: previous=%+v changed_at=%v", found.Previous, found.ChangedAt)
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
