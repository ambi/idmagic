package bootstrap

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 保持期間の境界は IdManagement のテストが確かめる。ここでは、掃除のバッチが IdManagement の
// 成果物ストアの期限切れの削除を呼ぶことを確かめる。
func TestRetentionSweepPurgesExpiredCSVArtifacts(t *testing.T) {
	ctx := context.Background()
	store := idmanagement.NewMemoryModule().CSVArtifacts
	metadata, err := store.PutCSVArtifact(ctx, "tenant-a", func(output io.Writer) error {
		_, err := io.WriteString(output, "preferred_username\nalice\n")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	deps := &Dependencies{IdManagement: idmanagement.Module{CSVArtifacts: store}}
	if err := RunRetentionSweepOnce(ctx, deps, time.Now().UTC().Add(10*365*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", metadata.Ref); err == nil {
		t.Fatal("保持期限を過ぎた成果物が残った")
	}
}

type recordingEventSink struct{ events []spec.DomainEvent }

func (s *recordingEventSink) Emit(_ context.Context, event spec.DomainEvent) error {
	s.events = append(s.events, event)
	return nil
}

//spec:covers EX-IDMANAGEMENT-044-01, REQ-IDMANAGEMENT-044: 保持期限の削除のバッチが、一覧の取得を待たずに、すべてのテナントで猶予期間を過ぎた削除予約の User を system と auto_purge で完全削除し、猶予期間内の User を残すこと。
func TestRetentionSweepPurgesUsersPastTheGracePeriod(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	tenants := tenancy.NewMemoryModule().TenantRepo
	users := idmanagement.NewMemoryModule().UserRepo
	pending := func(tenantID, id string, since time.Time) {
		t.Helper()
		if err := tenants.Save(ctx, &tenancydomain.Tenant{ID: tenantID, Realm: tenantID, Status: tenancydomain.TenantStatusActive}); err != nil {
			t.Fatal(err)
		}
		if err := users.Save(ctx, &userdomain.User{
			ID: id, TenantID: tenantID, PreferredUsername: id, PasswordHash: "hash",
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusPendingDeletion, StatusChangedAt: &since},
			CreatedAt: since, UpdatedAt: since,
		}); err != nil {
			t.Fatal(err)
		}
	}
	expiredSince := now.Add(-31 * 24 * time.Hour)
	pending("tenant-a", "expired-a", expiredSince)
	pending("tenant-b", "expired-b", expiredSince)
	pending("tenant-b", "within-grace", now.Add(-24*time.Hour))
	sink := &recordingEventSink{}
	deps := &Dependencies{
		Tenancy:      tenancy.Module{TenantRepo: tenants},
		IdManagement: idmanagement.Module{UserRepo: users},
		OAuth2:       oauth2.Module{EventSink: sink},
	}

	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"expired-a", "expired-b"} {
		if stored, err := users.FindBySubIncludingDeleted(ctx, id); err != nil || stored == nil || !stored.IsDeleted() {
			t.Fatalf("%s: stored=%+v err=%v, want deleted", id, stored, err)
		}
	}
	if stored, _ := users.FindBySub(ctx, "within-grace"); stored == nil || stored.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("within-grace=%+v, want pending_deletion", stored)
	}
	purged := map[string]bool{}
	for _, event := range sink.events {
		if deleted, ok := event.(*idmdomain.UserDeleted); ok {
			if deleted.ActorUserID != "system" || deleted.Reason != "auto_purge" {
				t.Fatalf("UserDeleted=%+v, want actor system and reason auto_purge", deleted)
			}
			purged[deleted.TenantID+"/"+deleted.TargetUserID] = true
		}
	}
	if len(purged) != 2 || !purged["tenant-a/expired-a"] || !purged["tenant-b/expired-b"] {
		t.Fatalf("UserDeleted targets=%v, want tenant-a/expired-a and tenant-b/expired-b", purged)
	}
}
