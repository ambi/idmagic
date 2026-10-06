package bootstrap

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers EX-IDMANAGEMENT-080-01: 保持期限の削除のバッチが、IdManagement の成果物ストアから保持期限を過ぎた成果物を消し、新しい成果物を残すこと。
func TestRetentionSweepDeletesCSVArtifactsPastRetention(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := idmmemory.NewCSVArtifactStore()
	ctx := context.Background()
	put := func(savedAt time.Time) string {
		t.Helper()
		store.Now = func() time.Time { return savedAt }
		metadata, err := store.PutCSVArtifact(ctx, "tenant-a", func(output io.Writer) error {
			_, err := io.WriteString(output, "preferred_username\nalice\n")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return metadata.Ref
	}
	expired := put(now.Add(-idmusecases.CSVArtifactRetention - time.Hour))
	fresh := put(now.Add(-time.Hour))

	deps := &Dependencies{IdManagement: idmanagement.Module{CSVArtifacts: store}}
	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", expired); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("保持期限を過ぎた成果物が残った: err=%v", err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", fresh); err != nil {
		t.Fatalf("新しい成果物が消えた: %v", err)
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
	tenants := tenancymemory.NewTenantRepository()
	users := usermemory.NewUserRepository()
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
