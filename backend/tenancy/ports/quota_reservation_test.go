package ports_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func groupsUsage(t *testing.T, repo tenantports.QuotaRepository, tenantID string) int {
	t.Helper()
	usage, err := repo.GetUsage(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Groups
}

//spec:covers REQ-TENANCY-050: 確定しなかった予約は加算を戻し、Keep した予約は加算を残す。上限の超過では予約を返さない。
func TestQuotaReservationReleasesUnlessKept(t *testing.T) {
	ctx := context.Background()
	const tenantID = "acme"
	repo := tenancy.NewMemoryModule().QuotaRepo
	limit := 1
	if err := repo.SetQuota(ctx, tenantID, &domain.TenantQuota{Groups: &limit}); err != nil {
		t.Fatal(err)
	}

	failed, err := tenantports.ReserveQuota(ctx, repo, tenantID, domain.ResourceGroups, 1)
	if err != nil {
		t.Fatalf("ReserveQuota: %v", err)
	}
	failed.Release(ctx)
	failed.Release(ctx) // 二度目は何も戻さない。
	if got := groupsUsage(t, repo, tenantID); got != 0 {
		t.Fatalf("usage after releasing an unkept reservation = %d, want 0", got)
	}

	created, err := tenantports.ReserveQuota(ctx, repo, tenantID, domain.ResourceGroups, 1)
	if err != nil {
		t.Fatalf("ReserveQuota after the release: %v", err)
	}
	created.Keep()
	created.Release(ctx)
	if got := groupsUsage(t, repo, tenantID); got != 1 {
		t.Fatalf("usage after a kept reservation = %d, want 1", got)
	}

	over, err := tenantports.ReserveQuota(ctx, repo, tenantID, domain.ResourceGroups, 1)
	if qErr, ok := errors.AsType[*domain.QuotaExceededError](err); !ok || over != nil {
		t.Fatalf("ReserveQuota over the limit = (%v, %v), want (nil, QuotaExceededError)", over, qErr)
	}
	over.Release(ctx) // 予約のない Release は何もしない。
	if got := groupsUsage(t, repo, tenantID); got != 1 {
		t.Fatalf("usage after a rejected reservation = %d, want 1", got)
	}
}

// 上限を強制しない構成では、予約は何も加算せず、戻す操作も何もしない。
func TestQuotaReservationWithoutRepositoryDoesNothing(t *testing.T) {
	ctx := context.Background()
	reservation, err := tenantports.ReserveQuota(ctx, nil, "acme", domain.ResourceGroups, 1)
	if err != nil || reservation == nil {
		t.Fatalf("ReserveQuota without a repository = (%v, %v), want a no-op reservation", reservation, err)
	}
	reservation.Release(ctx)
	reservation.Keep()
}
