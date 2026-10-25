package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-TENANCY-048: 保持期限バッチが実データをテナント別に計数し、Hard の使用量を保ち、Soft の超過だけを正式な監査経路へ送る。
func TestRetentionSweepReconcilesSoftQuotas(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	sink := &recordingEventSink{}
	deps := &Dependencies{Tenancy: tenancy.NewMemoryModule(), Audit: audit.NewMemoryModule(100), IdManagement: idmanagement.NewMemoryModule(), OAuth2: oauth2.Module{EventSink: sink}}
	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := deps.Tenancy.TenantRepo.Save(ctx, &domain.Tenant{ID: id, Realm: id, Status: domain.TenantStatusActive}); err != nil {
			t.Fatal(err)
		}
		if err := deps.Audit.AuditEventRepo.Append(ctx, &auditports.AuditEventRecord{ID: id, TenantID: id, Type: "TenantCreated", OccurredAt: now}); err != nil {
			t.Fatal(err)
		}
		if _, err := deps.IdManagement.CSVArtifacts.PutCSVArtifact(ctx, id, func(w io.Writer) error { _, err := io.WriteString(w, "abc"); return err }); err != nil {
			t.Fatal(err)
		}
	}
	if err := deps.Audit.AuditEventRepo.Append(ctx, &auditports.AuditEventRecord{ID: "second", TenantID: "tenant-a", Type: "TenantUpdated", OccurredAt: now}); err != nil {
		t.Fatal(err)
	}
	one, three := 1, 3
	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := deps.Tenancy.QuotaRepo.SetQuota(ctx, id, &domain.TenantQuota{AuditEventsRetained: &one, ExportArtifactsBytes: &three}); err != nil {
			t.Fatal(err)
		}
		if err := deps.Tenancy.QuotaRepo.CheckAndIncrement(ctx, id, domain.ResourceGroups, 2); err != nil {
			t.Fatal(err)
		}
	}
	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"tenant-a", "tenant-b"} {
		usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, id)
		wantCount := 1
		if id == "tenant-a" {
			wantCount = 2
		}
		if err != nil || usage.AuditEventsRetained != wantCount || usage.ExportArtifactsBytes != 3 || usage.Groups != 2 {
			t.Fatalf("%s: usage=%+v err=%v, want audit=%d bytes=3 groups=2", id, usage, err, wantCount)
		}
	}
	if len(sink.events) != 1 {
		t.Fatalf("events=%v, want one soft audit-count warning", sink.events)
	}
	wire, err := spec.MarshalDomainEvent(sink.events[0])
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(wire, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["type"] != "QuotaExceeded" || payload["tenantId"] != "tenant-a" || payload["resource"] != "audit_events_retained" || payload["hardLimit"] != false {
		t.Fatalf("warning payload=%s", wire)
	}
	records, err := deps.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: "tenant-a", Type: "QuotaExceeded"})
	if err != nil || len(records) != 1 || records[0].Payload["hardLimit"] != false {
		t.Fatalf("audit warnings=%+v err=%v", records, err)
	}
	// 期限切れの物理削除による減少も、同じバッチの測定に含める。
	if err := RunRetentionSweepOnce(ctx, deps, now.Add(10*365*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, "tenant-b")
	if err != nil || usage.ExportArtifactsBytes != 0 {
		t.Fatalf("after purge usage=%+v err=%v", usage, err)
	}
}

type quotaCountFailure struct {
	auditports.AuditEventRepository
}

func (r quotaCountFailure) Count(ctx context.Context, query auditports.AuditEventQuery) (int64, error) {
	if query.TenantID == "tenant-a" {
		return 0, errors.New("audit unavailable")
	}
	return r.AuditEventRepository.Count(ctx, query)
}

//spec:covers REQ-TENANCY-048: 測定失敗は以前の使用量を保ち、残りのテナントを照合してから失敗を返す。
func TestRetentionSweepKeepsUsageWhenMeasurementFails(t *testing.T) {
	ctx := context.Background()
	sink := &recordingEventSink{}
	deps := &Dependencies{Tenancy: tenancy.NewMemoryModule(), Audit: audit.NewMemoryModule(100), IdManagement: idmanagement.NewMemoryModule(), OAuth2: oauth2.Module{EventSink: sink}}
	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := deps.Tenancy.TenantRepo.Save(ctx, &domain.Tenant{ID: id, Realm: id, Status: domain.TenantStatusActive}); err != nil {
			t.Fatal(err)
		}
		if err := deps.Tenancy.QuotaRepo.SetSoftUsage(ctx, id, domain.SoftQuotaUsage{AuditEventsRetained: 12, ExportArtifactsBytes: 34}); err != nil {
			t.Fatal(err)
		}
	}
	deps.Audit.AuditEventRepo = quotaCountFailure{deps.Audit.AuditEventRepo}
	if err := RunRetentionSweepOnce(ctx, deps, time.Now()); err == nil {
		t.Fatal("measurement failure was ignored")
	}
	a, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, "tenant-a")
	if err != nil || a.AuditEventsRetained != 12 || a.ExportArtifactsBytes != 34 {
		t.Fatalf("failed tenant usage=%+v err=%v", a, err)
	}
	b, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, "tenant-b")
	if err != nil || b.AuditEventsRetained != 0 || b.ExportArtifactsBytes != 0 {
		t.Fatalf("healthy tenant usage=%+v err=%v", b, err)
	}
	if len(sink.events) != 0 {
		t.Fatalf("failure emitted warnings=%v", sink.events)
	}
}
