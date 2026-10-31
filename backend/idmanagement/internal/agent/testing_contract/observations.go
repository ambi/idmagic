package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	"github.com/ambi/idmagic/backend/idmanagement/agent/ports"
)

// ObservationFixture は観測の保存先と、その確認先になる Agent の保存先である。OtherTenantID は
// テナント境界を確かめる二つ目のテナントで、NewID は保存先の ID の形式に合う値を作る。
type ObservationFixture struct {
	Repository    ports.AgentObservationRepository
	Agents        ports.AgentRepository
	TenantID      string
	OtherTenantID string
	Agent         *domain.Agent
	MissingAgent  string
	NewID         func() string
}

// RunAgentObservationContract は、順序、決定の保持、並行する確認、Agent の削除、保持期限が
// すべての保存先で同じ結果になることを確かめる。
func RunAgentObservationContract(t *testing.T, newFixture func(t *testing.T) ObservationFixture) {
	t.Helper()
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	observation := func(f ObservationFixture, subject, name string, at time.Time) *domain.AgentObservation {
		return &domain.AgentObservation{
			ID: f.NewID(), TenantID: f.TenantID, Issuer: "https://discovery.example", Subject: subject,
			DisplayName: &name, Resources: []string{name}, ObservedAt: at, SourceTokenID: "token-" + name, CreatedAt: base,
		}
	}
	importOne := func(t *testing.T, f ObservationFixture, item *domain.AgentObservation) ports.ObservationImportCounts {
		t.Helper()
		counts, err := f.Repository.Import(context.Background(), f.TenantID, []*domain.AgentObservation{item}, 100)
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		return counts
	}
	only := func(t *testing.T, f ObservationFixture, subject string) *domain.AgentObservation {
		t.Helper()
		rows, err := f.Repository.ListPage(context.Background(), f.TenantID, ports.ObservationFilter{}, "", "", 100)
		if err != nil {
			t.Fatalf("ListPage: %v", err)
		}
		for _, row := range rows {
			if row.Subject == subject {
				return row
			}
		}
		t.Fatalf("%s が一覧に無い", subject)
		return nil
	}
	ctx := context.Background()

	t.Run("新しい観測だけが属性を更新し、再送と古い観測は変えない", func(t *testing.T) {
		f := newFixture(t)
		if counts := importOne(t, f, observation(f, "order", "first", base)); counts.Created != 1 {
			t.Fatalf("counts=%+v", counts)
		}
		if counts := importOne(t, f, observation(f, "order", "resend", base)); counts.Unchanged != 1 {
			t.Fatalf("再送の counts=%+v", counts)
		}
		if counts := importOne(t, f, observation(f, "order", "older", base.Add(-time.Minute))); counts.Unchanged != 1 {
			t.Fatalf("古い観測の counts=%+v", counts)
		}
		if got := only(t, f, "order"); *got.DisplayName != "first" || got.SourceTokenID != "token-first" {
			t.Fatalf("再送か古い観測が値を変えた: %+v", got)
		}
		if counts := importOne(t, f, observation(f, "order", "newer", base.Add(time.Minute))); counts.Updated != 1 {
			t.Fatalf("新しい観測の counts=%+v", counts)
		}
		got := only(t, f, "order")
		if *got.DisplayName != "newer" || got.SourceTokenID != "token-newer" || !got.ObservedAt.Equal(base.Add(time.Minute)) || got.Resources[0] != "newer" {
			t.Fatalf("新しい観測が反映されない: %+v", got)
		}
	})

	t.Run("同じバッチの同じ主体は最も新しい項目を残す", func(t *testing.T) {
		f := newFixture(t)
		counts, err := f.Repository.Import(ctx, f.TenantID, []*domain.AgentObservation{
			observation(f, "batch", "newest", base.Add(time.Minute)),
			observation(f, "batch", "oldest", base),
		}, 100)
		if err != nil {
			t.Fatal(err)
		}
		if counts.Created != 1 || counts.Unchanged != 1 {
			t.Fatalf("counts=%+v", counts)
		}
		if got := only(t, f, "batch"); *got.DisplayName != "newest" {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("上限を超えるバッチは一件も保存しない", func(t *testing.T) {
		f := newFixture(t)
		importOne(t, f, observation(f, "limit-0", "kept", base))
		before, err := f.Repository.Count(ctx, f.TenantID, ports.ObservationFilter{})
		if err != nil {
			t.Fatal(err)
		}
		limit := int(before) + 1
		_, err = f.Repository.Import(ctx, f.TenantID, []*domain.AgentObservation{
			observation(f, "limit-1", "a", base), observation(f, "limit-2", "b", base),
		}, limit)
		if !errors.Is(err, ports.ErrObservationLimitExceeded) {
			t.Fatalf("err=%v", err)
		}
		if after, _ := f.Repository.Count(ctx, f.TenantID, ports.ObservationFilter{}); after != before {
			t.Fatalf("拒否が保存を残した: %d → %d", before, after)
		}
		if _, err := f.Repository.Import(ctx, f.TenantID, []*domain.AgentObservation{observation(f, "limit-1", "a", base)}, limit); err != nil {
			t.Fatalf("上限ちょうどの保存: %v", err)
		}
	})

	t.Run("決定は再観測で変わらず、先の決定を上書きしない", func(t *testing.T) {
		f := newFixture(t)
		importOne(t, f, observation(f, "decide", "first", base))
		id := only(t, f, "decide").ID
		changed, err := f.Repository.Confirm(ctx, f.TenantID, id, f.Agent.ID, "actor", base)
		if err != nil || !changed {
			t.Fatalf("Confirm=(%v,%v)", changed, err)
		}
		if changed, err := f.Repository.Confirm(ctx, f.TenantID, id, f.Agent.ID, "actor-2", base); err != nil || changed {
			t.Fatalf("二度目の Confirm=(%v,%v)", changed, err)
		}
		if changed, err := f.Repository.Exclude(ctx, f.TenantID, id, "actor-2", base); err != nil || changed {
			t.Fatalf("確認後の Exclude=(%v,%v)", changed, err)
		}
		importOne(t, f, observation(f, "decide", "again", base.Add(time.Minute)))
		got := only(t, f, "decide")
		if got.ConfirmedAgentID == nil || *got.ConfirmedAgentID != f.Agent.ID || *got.DisplayName != "again" {
			t.Fatalf("再観測の後=%+v", got)
		}
		previous, err := f.Repository.Reopen(ctx, f.TenantID, id)
		if err != nil || previous == nil || previous.ConfirmedAgentID == nil {
			t.Fatalf("Reopen=(%+v,%v)", previous, err)
		}
		if got := only(t, f, "decide"); got.Decision() != domain.ObservationPending || got.DecidedAt != nil {
			t.Fatalf("再開の後=%+v", got)
		}
	})

	t.Run("存在しない Agent と別のテナントは確定しない", func(t *testing.T) {
		f := newFixture(t)
		importOne(t, f, observation(f, "tenant", "first", base))
		id := only(t, f, "tenant").ID
		if _, err := f.Repository.Confirm(ctx, f.TenantID, id, f.MissingAgent, "actor", base); !errors.Is(err, ports.ErrObservationAgentMissing) {
			t.Fatalf("err=%v", err)
		}
		if got, err := f.Repository.FindByID(ctx, f.OtherTenantID, id); err != nil || got != nil {
			t.Fatalf("別のテナントの FindByID=(%+v,%v)", got, err)
		}
		if got, err := f.Repository.Reopen(ctx, f.OtherTenantID, id); err != nil || got != nil {
			t.Fatalf("別のテナントの Reopen=(%+v,%v)", got, err)
		}
		if changed, err := f.Repository.Exclude(ctx, f.OtherTenantID, id, "actor", base); err != nil || changed {
			t.Fatalf("別のテナントの Exclude=(%v,%v)", changed, err)
		}
		if got, err := f.Repository.FindByID(ctx, f.TenantID, "not-a-uuid"); err != nil || got != nil {
			t.Fatalf("形式の不正な id=(%+v,%v)", got, err)
		}
	})

	t.Run("Agent の削除が確認を解除する", func(t *testing.T) {
		f := newFixture(t)
		importOne(t, f, observation(f, "deleted", "first", base))
		id := only(t, f, "deleted").ID
		if _, err := f.Repository.Confirm(ctx, f.TenantID, id, f.Agent.ID, "actor", base); err != nil {
			t.Fatal(err)
		}
		if err := f.Agents.Delete(ctx, f.TenantID, f.Agent.ID); err != nil {
			t.Fatal(err)
		}
		if got := only(t, f, "deleted"); got.ConfirmedAgentID != nil || got.Decision() != domain.ObservationPending {
			t.Fatalf("削除の後=%+v", got)
		}
	})

	t.Run("保持期限は確認していない観測だけを消す", func(t *testing.T) {
		f := newFixture(t)
		for _, subject := range []string{"old-pending", "old-excluded", "old-confirmed"} {
			importOne(t, f, observation(f, subject, "x", base))
		}
		importOne(t, f, observation(f, "boundary", "x", base.Add(time.Hour)))
		if _, err := f.Repository.Exclude(ctx, f.TenantID, only(t, f, "old-excluded").ID, "actor", base); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Repository.Confirm(ctx, f.TenantID, only(t, f, "old-confirmed").ID, f.Agent.ID, "actor", base); err != nil {
			t.Fatal(err)
		}
		deleted, err := f.Repository.PurgeExpired(ctx, base.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if deleted < 2 {
			t.Fatalf("deleted=%d", deleted)
		}
		rows, err := f.Repository.ListPage(ctx, f.TenantID, ports.ObservationFilter{}, "", "", 100)
		if err != nil {
			t.Fatal(err)
		}
		subjects := map[string]bool{}
		for _, row := range rows {
			subjects[row.Subject] = true
		}
		if subjects["old-pending"] || subjects["old-excluded"] || !subjects["old-confirmed"] || !subjects["boundary"] {
			t.Fatalf("残った観測=%v", subjects)
		}
		confirmed, err := f.Repository.ListConfirmed(ctx, f.TenantID)
		if err != nil || len(confirmed) != 1 {
			t.Fatalf("ListConfirmed=(%v,%v)", confirmed, err)
		}
	})

	t.Run("決定で絞り込み、キーセットで双方向にたどる", func(t *testing.T) {
		f := newFixture(t)
		for _, subject := range []string{"page-a", "page-b", "page-c"} {
			importOne(t, f, observation(f, subject, "x", base))
		}
		excluded := only(t, f, "page-b")
		if _, err := f.Repository.Exclude(ctx, f.TenantID, excluded.ID, "actor", base); err != nil {
			t.Fatal(err)
		}
		pending := ports.ObservationFilter{Decision: domain.ObservationPending}
		first, err := f.Repository.ListPage(ctx, f.TenantID, pending, "", "", 1)
		if err != nil || len(first) != 1 || first[0].Subject != "page-a" {
			t.Fatalf("first=(%v,%v)", first, err)
		}
		next, err := f.Repository.ListPage(ctx, f.TenantID, pending, first[0].Subject, first[0].ID, 10)
		if err != nil || len(next) != 1 || next[0].Subject != "page-c" {
			t.Fatalf("next=(%v,%v)", next, err)
		}
		before, err := f.Repository.ListPageBefore(ctx, f.TenantID, pending, next[0].Subject, next[0].ID, 10)
		if err != nil || len(before) != 1 || before[0].Subject != "page-a" {
			t.Fatalf("before=(%v,%v)", before, err)
		}
		if count, err := f.Repository.Count(ctx, f.TenantID, ports.ObservationFilter{Decision: domain.ObservationExcluded}); err != nil || count != 1 {
			t.Fatalf("count=(%d,%v)", count, err)
		}
	})
}
