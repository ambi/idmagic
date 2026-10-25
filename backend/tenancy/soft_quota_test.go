package tenancy_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-TENANCY-047, REQ-TENANCY-048: 公開操作で既定値、上書き、上限ちょうど、超過と警告の送信表現を確かめる。
func TestReconcileSoftQuota(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit *int
		usage domain.SoftQuotaUsage
		want  []string
	}{
		{name: "default boundary", usage: domain.SoftQuotaUsage{AuditEventsRetained: 1000000, ExportArtifactsBytes: 1 << 30}},
		{name: "above defaults", usage: domain.SoftQuotaUsage{AuditEventsRetained: 1000001, ExportArtifactsBytes: 1<<30 + 1}, want: []string{"audit_events_retained", "export_artifacts_bytes"}},
		{name: "override boundary", limit: new(2), usage: domain.SoftQuotaUsage{AuditEventsRetained: 2, ExportArtifactsBytes: 2}},
		{name: "below override", limit: new(2), usage: domain.SoftQuotaUsage{AuditEventsRetained: 1, ExportArtifactsBytes: 1}},
		{name: "zero override", limit: new(0), usage: domain.SoftQuotaUsage{AuditEventsRetained: 1}, want: []string{"audit_events_retained"}},
		{name: "zero usage", limit: new(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			m := tenancy.NewMemoryModule()
			if err := m.QuotaRepo.SetQuota(ctx, "tenant", &domain.TenantQuota{AuditEventsRetained: tc.limit, ExportArtifactsBytes: tc.limit}); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
			var events []spec.DomainEvent
			emit := func(e spec.DomainEvent) { events = append(events, e) }
			for attempt := range 2 {
				if err := m.ReconcileSoftQuota(ctx, "tenant", tc.usage, now, emit); err != nil {
					t.Fatal(err)
				}
				if len(events) != (attempt+1)*len(tc.want) {
					t.Fatalf("events=%v, want %v each attempt", events, tc.want)
				}
			}
			for i, resource := range tc.want {
				wire, err := spec.MarshalDomainEvent(events[i])
				if err != nil {
					t.Fatal(err)
				}
				var payload map[string]any
				if err := json.Unmarshal(wire, &payload); err != nil {
					t.Fatal(err)
				}
				if payload["tenantId"] != "tenant" || payload["resource"] != resource || payload["hardLimit"] != false || payload["occurredAt"] != "2026-10-25T00:00:00Z" {
					t.Fatalf("wire=%s", wire)
				}
			}
			usage, err := m.QuotaRepo.GetUsage(ctx, "tenant")
			if err != nil || usage.AuditEventsRetained != tc.usage.AuditEventsRetained || usage.ExportArtifactsBytes != tc.usage.ExportArtifactsBytes {
				t.Fatalf("usage=%+v err=%v", usage, err)
			}
		})
	}
}

type failingSoftQuotaRepository struct{ ports.QuotaRepository }

func (failingSoftQuotaRepository) SetSoftUsage(context.Context, string, domain.SoftQuotaUsage) error {
	return errors.New("store unavailable")
}

//spec:covers REQ-TENANCY-048: 保存の失敗で使用量と警告を残さない。
func TestReconcileSoftQuotaDoesNotWarnWhenSavingFails(t *testing.T) {
	ctx := context.Background()
	m := tenancy.NewMemoryModule()
	repository := m.QuotaRepo
	m.QuotaRepo = failingSoftQuotaRepository{repository}
	var events []spec.DomainEvent
	err := m.ReconcileSoftQuota(ctx, "tenant", domain.SoftQuotaUsage{AuditEventsRetained: 1000001}, time.Now(), func(e spec.DomainEvent) { events = append(events, e) })
	usage, readErr := repository.GetUsage(ctx, "tenant")
	if err == nil || readErr != nil || *usage != (domain.TenantUsage{}) || len(events) != 0 {
		t.Fatalf("err=%v usage=%+v readErr=%v events=%v", err, usage, readErr, events)
	}
}
