package testing_contract

import (
	"context"
	"sync"
	"testing"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func RunSoftQuota(t *testing.T, newFixture NewQuotaFixture) {
	t.Helper()
	for _, resource := range []string{"audit_events_retained", "export_artifacts_bytes"} {
		t.Run(resource, func(t *testing.T) {
			ctx := context.Background()
			f := newFixture(t)
			one := 1
			if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{AuditEventsRetained: &one, ExportArtifactsBytes: &one}); err != nil {
				t.Fatal(err)
			}
			read := func(u *domain.TenantUsage) int {
				if resource == "audit_events_retained" {
					return u.AuditEventsRetained
				}
				return u.ExportArtifactsBytes
			}
			for want := 1; want <= 2; want++ {
				if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, resource, 1); err != nil {
					t.Fatalf("soft increment to %d: %v", want, err)
				}
				requireUsage(ctx, t, f, read, want)
			}
			zero := 0
			if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{AuditEventsRetained: &zero, ExportArtifactsBytes: &zero}); err != nil {
				t.Fatal(err)
			}
			const large = 3 << 30
			if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, resource, large); err != nil {
				t.Fatalf("increment above zero override and 2 GiB: %v", err)
			}
			requireUsage(ctx, t, f, read, large+2)
			if err := f.Repository.Decrement(ctx, f.TenantID, resource, large+10); err != nil {
				t.Fatal(err)
			}
			requireUsage(ctx, t, f, read, 0)
			for _, operation := range []func(context.Context, string, string, int) error{f.Repository.CheckAndIncrement, f.Repository.Decrement} {
				if err := operation(ctx, f.TenantID, resource, -1); err == nil {
					t.Fatal("negative soft delta accepted")
				}
			}
			requireUsage(ctx, t, f, read, 0)
			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() {
					if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, resource, 1); err != nil {
						t.Errorf("concurrent increment: %v", err)
					}
				})
			}
			wg.Wait()
			requireUsage(ctx, t, f, read, 8)
			other := newFixture(t)
			requireUsage(ctx, t, other, read, 0)
			largeLimit := large + 1
			if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{AuditEventsRetained: &largeLimit, ExportArtifactsBytes: &largeLimit}); err != nil {
				t.Fatal(err)
			}
			quota, err := f.Repository.GetQuota(ctx, f.TenantID)
			if err != nil || quota.EffectiveLimit(resource) != largeLimit {
				t.Fatalf("large override: quota=%+v err=%v", quota, err)
			}
			if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceGroups, 1); err != nil {
				t.Fatal(err)
			}
			measured := domain.SoftQuotaUsage{AuditEventsRetained: 3, ExportArtifactsBytes: large}
			if err := f.Repository.SetSoftUsage(ctx, f.TenantID, measured); err != nil {
				t.Fatal(err)
			}
			if err := f.Repository.SetSoftUsage(ctx, f.TenantID, domain.SoftQuotaUsage{AuditEventsRetained: -1}); err == nil {
				t.Fatal("negative snapshot accepted")
			}
			usage, err := f.Repository.GetUsage(ctx, f.TenantID)
			if err != nil || usage.AuditEventsRetained != 3 || usage.ExportArtifactsBytes != large || usage.Groups != 1 {
				t.Fatalf("snapshot usage=%+v err=%v", usage, err)
			}
			// 読み戻した値の変更は保存した値へ伝播しない。
			usage.AuditEventsRetained = 99
			requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.AuditEventsRetained }, 3)
			requireUsage(ctx, t, other, read, 0)
			if err := f.Repository.SetSoftUsage(ctx, f.TenantID, domain.SoftQuotaUsage{}); err != nil {
				t.Fatal(err)
			}
			requireUsage(ctx, t, f, read, 0)
		})
	}
}
