package testing_contract

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// RunMcpToolCatalogs はカタログの置き換え、テナントの分離、削除を両保存先で確認する。
func RunMcpToolCatalogs(t *testing.T, repo ports.McpToolCatalogRepository, tenantID, otherTenantID, serverID string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if found, err := repo.Find(ctx, tenantID, serverID); err != nil || found != nil {
		t.Fatalf("before any report: %+v %v", found, err)
	}
	defer func() {
		if t.Failed() {
			return
		}
		// 最初のカタログには比べる相手がないので、変更前の一覧を作らない。
		fresh := &domain.McpToolCatalog{TenantID: tenantID, ResourceServerID: serverID, Complete: true, ReceivedAt: now, Tools: []domain.CatalogTool{{Name: "a"}}}
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
	first := &domain.McpToolCatalog{
		TenantID: tenantID, ResourceServerID: serverID, Complete: true, ReceivedAt: now,
		Tools: []domain.CatalogTool{{Name: "read_document", Title: "閲覧", Description: "文書を読む", Arguments: []string{"document_id"}}, {Name: "delete_document"}},
	}
	if err := repo.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	second := &domain.McpToolCatalog{
		TenantID: tenantID, ResourceServerID: serverID, Complete: false, ReceivedAt: now.Add(time.Minute),
		Tools: []domain.CatalogTool{{Name: "write_document"}},
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	found, err := repo.Find(ctx, tenantID, serverID)
	if err != nil || found == nil || found.Complete || len(found.Tools) != 1 || found.Tools[0].Name != "write_document" || !found.ReceivedAt.Equal(second.ReceivedAt) {
		t.Fatalf("replaced report=%+v err=%v", found, err)
	}
	// 内容の変わったカタログは、直前の一覧と complete を残し、受信の時刻を変更の時刻にする (REQ-OAUTH2-086)。
	if found.Previous == nil || !reflect.DeepEqual(*found.Previous, domain.CatalogToolList{Tools: first.Tools, Complete: true}) || !found.ChangedAt.Equal(second.ReceivedAt) {
		t.Fatalf("a changed report did not keep the previous list: previous=%+v changed_at=%v", found.Previous, found.ChangedAt)
	}
	// 同じ内容のカタログは、受信の時刻と complete だけを変え、変更前の一覧を保つ。入力の Previous は読まない。
	same := &domain.McpToolCatalog{
		TenantID: tenantID, ResourceServerID: serverID, Complete: true, ReceivedAt: now.Add(2 * time.Minute),
		Tools:    []domain.CatalogTool{{Name: "write_document"}},
		Previous: &domain.CatalogToolList{Tools: []domain.CatalogTool{{Name: "forged"}}}, ChangedAt: now.Add(time.Hour),
	}
	if err := repo.Save(ctx, same); err != nil {
		t.Fatal(err)
	}
	found, err = repo.Find(ctx, tenantID, serverID)
	if err != nil || found == nil || !found.Complete || !found.ReceivedAt.Equal(same.ReceivedAt) {
		t.Fatalf("the same report was not saved: %+v %v", found, err)
	}
	if found.Previous == nil || !reflect.DeepEqual(found.Previous.Tools, first.Tools) || !found.ChangedAt.Equal(second.ReceivedAt) {
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
