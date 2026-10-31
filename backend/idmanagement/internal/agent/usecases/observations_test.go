package usecases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/internal/agent/db_memory"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type stubWorkloadSubjects struct {
	agentIDs []string
	err      error
}

func (s stubWorkloadSubjects) AgentsForSubject(context.Context, string, string, string) ([]string, error) {
	return s.agentIDs, s.err
}

type observationFixture struct {
	ctx    context.Context
	deps   AdminAgentDeps
	mu     *sync.Mutex
	events *[]spec.DomainEvent
	now    time.Time
}

func newObservationUsecaseFixture(t *testing.T, agentIDs ...string) observationFixture {
	t.Helper()
	agents := agentmemory.NewAgentRepository()
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, id := range agentIDs {
		if err := agents.Save(ctx, &agentdomain.Agent{ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner", Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	events := &[]spec.DomainEvent{}
	mu := &sync.Mutex{}
	return observationFixture{
		ctx: ctx, mu: mu, events: events, now: now,
		deps: AdminAgentDeps{
			AgentRepo: agents, Observations: agentmemory.NewAgentObservationRepository(agents),
			WorkloadSubjects: stubWorkloadSubjects{},
			Emit: func(event spec.DomainEvent) error {
				mu.Lock()
				defer mu.Unlock()
				*events = append(*events, event)
				return nil
			},
		},
	}
}

func (f observationFixture) importOne(t *testing.T, subject string, at time.Time) string {
	t.Helper()
	if _, err := ImportObservations(f.ctx, f.deps, "actor", "token", []*agentdomain.AgentObservation{{Issuer: "https://i.example", Subject: subject, ObservedAt: at}}, f.now); err != nil {
		t.Fatal(err)
	}
	rows, err := f.deps.Observations.ListPage(f.ctx, tenancydomain.DefaultTenantID, agentports.ObservationFilter{}, "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Subject == subject {
			return row.ID
		}
	}
	t.Fatalf("%s が保存されない", subject)
	return ""
}

//spec:covers REQ-IDMANAGEMENT-096: 変更があるバッチだけ件数のイベントを一件発行し、値を載せない
func TestImportObservationsEmitsOnlyWhenSomethingChanged(t *testing.T) {
	f := newObservationUsecaseFixture(t)
	f.importOne(t, "svc", f.now)
	if len(*f.events) != 1 {
		t.Fatalf("events=%v", *f.events)
	}
	event, ok := (*f.events)[0].(*idmdomain.AgentObservationsImported)
	if !ok || event.Created != 1 || event.SourceTokenID != "token" || event.ActorUserID != "actor" {
		t.Fatalf("event=%+v", (*f.events)[0])
	}
	f.importOne(t, "svc", f.now)
	if len(*f.events) != 1 {
		t.Fatalf("変更のない再送がイベントを出した: %v", *f.events)
	}
}

//spec:covers REQ-IDMANAGEMENT-096: 一件でも不正なら一件も保存せず、イベントも出さない
func TestImportObservationsRejectsWholeBatchOnInvalidItem(t *testing.T) {
	f := newObservationUsecaseFixture(t)
	_, err := ImportObservations(f.ctx, f.deps, "actor", "token", []*agentdomain.AgentObservation{
		{Issuer: "https://i.example", Subject: "good", ObservedAt: f.now},
		{Issuer: "https://i.example", Subject: "bad"},
	}, f.now)
	if !errors.Is(err, agentdomain.ErrInvalidObservation) {
		t.Fatalf("err=%v", err)
	}
	if count, _ := f.deps.Observations.Count(f.ctx, tenancydomain.DefaultTenantID, agentports.ObservationFilter{}); count != 0 || len(*f.events) != 0 {
		t.Fatalf("count=%d events=%v", count, *f.events)
	}
}

//spec:covers REQ-IDMANAGEMENT-098: 候補の材料を読めなければ、候補を欠いた一覧を返さない
func TestViewObservationsFailsClosed(t *testing.T) {
	f := newObservationUsecaseFixture(t)
	f.importOne(t, "svc", f.now)
	rows, _ := f.deps.Observations.ListPage(f.ctx, tenancydomain.DefaultTenantID, agentports.ObservationFilter{}, "", "", 10)

	failing := f.deps
	boom := errors.New("workload store down")
	failing.WorkloadSubjects = stubWorkloadSubjects{err: boom}
	if _, err := ViewObservations(f.ctx, failing, rows, f.now); !errors.Is(err, boom) {
		t.Errorf("err=%v", err)
	}
	unwired := f.deps
	unwired.WorkloadSubjects = nil
	if _, err := ViewObservations(f.ctx, unwired, rows, f.now); err == nil {
		t.Error("未配線で候補を欠いた一覧を返した")
	}
}

//spec:covers REQ-IDMANAGEMENT-099: 候補でない Agent の確認をイベントで区別する
func TestConfirmRecordsWhetherAgentWasACandidate(t *testing.T) {
	f := newObservationUsecaseFixture(t, "candidate", "stranger")
	f.deps.WorkloadSubjects = stubWorkloadSubjects{agentIDs: []string{"candidate"}}
	reasons := func(id, agentID string) []string {
		t.Helper()
		if _, err := ConfirmAgentMatch(f.ctx, f.deps, "actor", id, agentID, f.now); err != nil {
			t.Fatal(err)
		}
		event, ok := (*f.events)[len(*f.events)-1].(*idmdomain.AgentObservationConfirmed)
		if !ok {
			t.Fatalf("event=%+v", (*f.events)[len(*f.events)-1])
		}
		return event.Reasons
	}
	if got := reasons(f.importOne(t, "one", f.now), "candidate"); len(got) != 1 || got[0] != "workload_binding" {
		t.Errorf("候補の根拠=%v", got)
	}
	if got := reasons(f.importOne(t, "two", f.now), "stranger"); len(got) != 0 {
		t.Errorf("候補でない Agent の根拠=%v", got)
	}
}

//spec:covers REQ-IDMANAGEMENT-099: 並行する確認は一方だけが確定し、負けた側は先の決定を上書きしない
func TestConcurrentConfirmDecidesOnce(t *testing.T) {
	f := newObservationUsecaseFixture(t, "a", "b")
	id := f.importOne(t, "svc", f.now)
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i, agentID := range []string{"a", "b"} {
		wg.Go(func() { _, results[i] = ConfirmAgentMatch(f.ctx, f.deps, "actor", id, agentID, f.now) })
	}
	wg.Wait()
	winners, losers := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrObservationAlreadyDecided):
			losers++
		default:
			t.Fatalf("err=%v", err)
		}
	}
	if winners != 1 || losers != 1 {
		t.Fatalf("results=%v", results)
	}
	confirmed := 0
	for _, event := range *f.events {
		if _, ok := event.(*idmdomain.AgentObservationConfirmed); ok {
			confirmed++
		}
	}
	if confirmed != 1 {
		t.Fatalf("確認のイベント=%d", confirmed)
	}
}

//spec:covers REQ-IDMANAGEMENT-100: 同じ除外と未決定の再開はイベントを出さない
func TestExcludeAndReopenAreIdempotent(t *testing.T) {
	f := newObservationUsecaseFixture(t, "a")
	id := f.importOne(t, "svc", f.now)
	baseline := len(*f.events)
	if _, err := ReopenObservation(f.ctx, f.deps, "actor", id, f.now); err != nil {
		t.Fatal(err)
	}
	if len(*f.events) != baseline {
		t.Fatalf("未決定の再開がイベントを出した: %v", *f.events)
	}
	for range 2 {
		if _, err := ExcludeObservation(f.ctx, f.deps, "actor", id, f.now); err != nil {
			t.Fatal(err)
		}
	}
	if len(*f.events) != baseline+1 {
		t.Fatalf("除外のイベント=%v", *f.events)
	}
	if _, err := ConfirmAgentMatch(f.ctx, f.deps, "actor", id, "a", f.now); !errors.Is(err, ErrObservationAlreadyDecided) {
		t.Fatalf("除外済みの確認 err=%v", err)
	}
	if _, err := ExcludeObservation(f.ctx, f.deps, "actor", "missing", f.now); !errors.Is(err, ErrObservationNotFound) {
		t.Fatalf("未知の観測の除外 err=%v", err)
	}
	if _, err := ReopenObservation(f.ctx, f.deps, "actor", "missing", f.now); !errors.Is(err, ErrObservationNotFound) {
		t.Fatalf("未知の観測の再開 err=%v", err)
	}
}
