package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-TENANCY-049: 保持期限バッチが、保存に失敗した作成で増えたままの使用量と減りすぎた使用量を、テナントごとに使用量を占める User の件数へ戻し、ほかのリソースの使用量を変えない。
func TestRetentionSweepRecountsUserUsage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	deps := &Dependencies{Tenancy: tenancy.NewMemoryModule(), Audit: audit.NewMemoryModule(100), IdManagement: idmanagement.NewMemoryModule(), OAuth2: oauth2.Module{EventSink: &recordingEventSink{}}}
	saveUser := func(tenantID, id string, lifecycle userdomain.UserLifecycle) {
		t.Helper()
		if err := deps.IdManagement.UserRepo.Save(ctx, &userdomain.User{
			ID: id, TenantID: tenantID, PreferredUsername: id, PasswordHash: "hash", Lifecycle: lifecycle, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := deps.Tenancy.TenantRepo.Save(ctx, &tenancydomain.Tenant{ID: id, Realm: id, Status: tenancydomain.TenantStatusActive}); err != nil {
			t.Fatal(err)
		}
		if err := deps.Tenancy.QuotaRepo.CheckAndIncrement(ctx, id, tenancydomain.ResourceGroups, 2); err != nil {
			t.Fatal(err)
		}
	}
	// tenant-a: 二人が使用量を占め、完全削除を終えた Tombstone は占めない。User の保存に失敗した作成の分だけ使用量が多い。
	withinGrace := now.Add(-time.Hour)
	saveUser("tenant-a", "alice", userdomain.UserLifecycle{Status: idmdomain.UserStatusActive})
	saveUser("tenant-a", "bob", userdomain.UserLifecycle{Status: idmdomain.UserStatusPendingDeletion, StatusChangedAt: &withinGrace})
	saveUser("tenant-a", "carol", userdomain.UserLifecycle{Status: idmdomain.UserStatusDeleted})
	if err := deps.Tenancy.QuotaRepo.CheckAndIncrement(ctx, "tenant-a", tenancydomain.ResourceUsers, 3); err != nil {
		t.Fatal(err)
	}
	// tenant-b: 完全削除の再実行が減算を繰り返し、使用量が一人分少ない。
	saveUser("tenant-b", "dave", userdomain.UserLifecycle{Status: idmdomain.UserStatusActive})

	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	for tenantID, wantUsers := range map[string]int{"tenant-a": 2, "tenant-b": 1} {
		usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, tenantID)
		if err != nil {
			t.Fatal(err)
		}
		if usage.Users != wantUsers || usage.Groups != 2 {
			t.Fatalf("%s: usage users=%d groups=%d, want users=%d groups=2", tenantID, usage.Users, usage.Groups, wantUsers)
		}
	}
}

type holdingCountFailure struct {
	userports.UserRepository
}

func (r holdingCountFailure) CountHoldingQuota(ctx context.Context, tenantID string) (int64, error) {
	if tenantID == "tenant-a" {
		return 0, errors.New("users unavailable")
	}
	return r.UserRepository.CountHoldingQuota(ctx, tenantID)
}

//spec:covers REQ-TENANCY-049: 件数の測定に失敗したテナントの使用量を保ち、残りのテナントを再集計してからバッチの失敗を返す。
func TestRetentionSweepKeepsUserUsageWhenTheCountFails(t *testing.T) {
	ctx := context.Background()
	deps := &Dependencies{Tenancy: tenancy.NewMemoryModule(), Audit: audit.NewMemoryModule(100), IdManagement: idmanagement.NewMemoryModule(), OAuth2: oauth2.Module{EventSink: &recordingEventSink{}}}
	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := deps.Tenancy.TenantRepo.Save(ctx, &tenancydomain.Tenant{ID: id, Realm: id, Status: tenancydomain.TenantStatusActive}); err != nil {
			t.Fatal(err)
		}
		if err := deps.Tenancy.QuotaRepo.CheckAndIncrement(ctx, id, tenancydomain.ResourceUsers, 4); err != nil {
			t.Fatal(err)
		}
	}
	deps.IdManagement.UserRepo = holdingCountFailure{deps.IdManagement.UserRepo}

	if err := RunRetentionSweepOnce(ctx, deps, time.Now()); err == nil {
		t.Fatal("count failure was ignored")
	}
	for tenantID, wantUsers := range map[string]int{"tenant-a": 4, "tenant-b": 0} {
		usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, tenantID)
		if err != nil || usage.Users != wantUsers {
			t.Fatalf("%s: usage=%+v err=%v, want users=%d", tenantID, usage, err, wantUsers)
		}
	}
}
