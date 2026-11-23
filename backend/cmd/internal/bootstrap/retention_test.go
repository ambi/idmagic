package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/idempotency"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SYSTEM-036: 正式なメモリの組み立てで受付記録を作り、retention sweep が期限の直前には残し、期限に達したら削除する。
func TestRetentionSweepPurgesIdempotencyRecords(t *testing.T) {
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	first, _, err := deps.Idempotency.Claim(t.Context(), "tenant-a", "admin", "key", "fingerprint", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.Idempotency.Complete(t.Context(), first, idempotency.Response{Status: 201, Body: []byte("secret")}); err != nil {
		t.Fatal(err)
	}
	if err := RunRetentionSweepOnce(t.Context(), deps, first.ExpiresAt.Add(-time.Microsecond)); err != nil {
		t.Fatal(err)
	}
	found, acquired, err := deps.Idempotency.Claim(t.Context(), "tenant-a", "admin", "key", "fingerprint", now)
	if err != nil || acquired || found.ID != first.ID {
		t.Fatalf("live record acquired=%v error=%v", acquired, err)
	}
	if err := RunRetentionSweepOnce(t.Context(), deps, first.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	// 照会に過去の時刻を使い、Claim 自体の期限切れ置換ではなく sweep の削除を観測する。
	_, acquired, err = deps.Idempotency.Claim(t.Context(), "tenant-a", "admin", "key", "fingerprint", now)
	if err != nil || !acquired {
		t.Fatalf("purged record acquired=%v error=%v", acquired, err)
	}
}

// 保持期間の境界は IdManagement のテストが確かめる。ここでは、掃除のバッチが IdManagement の
// 成果物ストアの期限切れの削除を呼ぶことを確かめる。
//
//spec:covers REQ-IDMANAGEMENT-080: 本番と同じ組み立ての Batch が期限切れ入力のジョブを取り消し、入力を削除する。
func TestRetentionSweepPurgesExpiredCSVArtifacts(t *testing.T) {
	ctx := context.Background()
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	store := deps.IdManagement.CSVArtifacts
	metadata, err := store.PutCSVArtifact(ctx, "tenant-a", func(output io.Writer) error {
		_, err := io.WriteString(output, "preferred_username\nalice\n")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]string{"artifact_ref": metadata.Ref})
	if err != nil {
		t.Fatal(err)
	}
	enqueuer := jobs.NewEnqueuer(deps.Jobs.Repo, deps.Tenancy.QuotaRepo, deps.NewEmitFunc(logging.Default()))
	job, err := enqueuer.Enqueue(ctx, jobsports.EnqueueInput{TenantID: "tenant-a", Kind: jobsdomain.KindUserImportPreview, Params: params}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	if err := RunRetentionSweepOnce(ctx, deps, time.Now().UTC().Add(10*365*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", metadata.Ref); err == nil {
		t.Fatal("保持期限を過ぎた成果物が残った")
	}
	if stored, err := deps.Jobs.Repo.Get(ctx, job.ID); err != nil || stored.Status != jobsdomain.StatusCanceled {
		t.Fatalf("期限切れ入力のジョブ=%+v err=%v", stored, err)
	}
	if events, err := deps.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: "tenant-a", Type: "JobCanceled"}); err != nil || len(events) != 1 || events[0].Payload["jobId"] != job.ID {
		t.Fatalf("JobCanceled=%v err=%v", events, err)
	}
	if usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, "tenant-a"); err != nil || usage.ActiveJobs != 0 {
		t.Fatalf("usage=%+v err=%v", usage, err)
	}
}

// 期間の全種類は Audit のテストが確かめる。ここでは、本番と同じメモリの組み立てから掃除のバッチが
// Audit の掃引へ届き、種類別の期間とデフォルトの期間の両側を区別することを確かめる。
//
//spec:covers REQ-AUDIT-013: 保持期間の掃引のバッチが、期間を過ぎた監査の記録を消し、期間の内側の記録を残す。
func TestRetentionSweepPurgesAuditEventsPastTheirPeriod(t *testing.T) {
	ctx := context.Background()
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	seed := []struct {
		id   string
		typ  string
		age  time.Duration
		kept bool
	}{
		{"failure-inside", "AuthenticationFailed", 29 * day, true},
		{"failure-outside", "AuthenticationFailed", 31 * day, false},
		{"impersonation-inside", "SessionImpersonationStarted", 364 * day, true},
		{"impersonation-outside", "SessionImpersonationStarted", 366 * day, false},
		{"admin-inside", "UserUpdated", 364 * day, true},
		{"admin-outside", "UserUpdated", 366 * day, false},
	}
	for _, s := range seed {
		if err := deps.Audit.AuditEventRepo.Append(ctx, &auditports.AuditEventRecord{
			ID: s.id, TenantID: tenancydomain.DefaultTenantID, Type: s.typ, OccurredAt: now.Add(-s.age),
			Payload: map[string]any{"tenantId": tenancydomain.DefaultTenantID},
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	for _, s := range seed {
		got, err := deps.Audit.AuditEventRepo.FindByID(ctx, s.id)
		if err != nil {
			t.Fatal(err)
		}
		if kept := got != nil; kept != s.kept {
			t.Errorf("%s kept=%v, want %v", s.id, kept, s.kept)
		}
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
