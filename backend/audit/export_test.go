package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/prometheus/client_golang/prometheus"
)

type exportFixture struct {
	module Module
	deps   ExportDeps
	now    time.Time
	events []spec.DomainEvent
}

func newExportFixture(tb testing.TB, count int) *exportFixture {
	tb.Helper()
	f := &exportFixture{module: NewMemoryModule(count + 100), now: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)}
	repo := jobs.NewJobRepository()
	emit := func(event spec.DomainEvent) { f.events = append(f.events, event) }
	quotas := tenancy.NewMemoryModule().QuotaRepo
	activeJobs := 100
	if err := quotas.SetQuota(context.Background(), "a", &tenantdomain.TenantQuota{ActiveJobs: &activeJobs}); err != nil {
		tb.Fatal(err)
	}
	f.deps = ExportDeps{Events: f.module.AuditEventRepo, Artifacts: f.module.ExportArtifacts, Jobs: repo, Enqueuer: jobs.NewEnqueuer(repo, quotas, emit), Quotas: quotas, Codec: f.module.ExportCodec, Emit: emit, Now: func() time.Time { return f.now }}
	for i := range count {
		if err := f.deps.Events.Append(context.Background(), &ports.AuditEventRecord{ID: fmt.Sprintf("%08d", i), TenantID: "a", Type: "UserAuthenticated", OccurredAt: f.now.Add(-time.Minute), Payload: map[string]any{"userId": "alice"}}); err != nil {
			tb.Fatal(err)
		}
	}
	return f
}

func (f *exportFixture) start(tb testing.TB, policy domain.ExportPolicy) *jobsdomain.Job {
	tb.Helper()
	view, err := StartExport(context.Background(), f.deps, ports.ExportParams{Query: ports.AuditEventQuery{TenantID: "a"}, RequestedBy: "owner", Format: "json", Policy: policy})
	if err != nil {
		tb.Fatal(err)
	}
	claimed, err := f.deps.Jobs.ClaimBatch(context.Background(), "worker", jobsdomain.LaneBulk, 1, time.Minute, f.now)
	if err != nil || len(claimed) != 1 || claimed[0].ID != view.ID {
		tb.Fatalf("claim=%v err=%v", claimed, err)
	}
	return claimed[0]
}

func (f *exportFixture) complete(tb testing.TB, job *jobsdomain.Job) ports.ExportArtifact {
	tb.Helper()
	result, err := usecases.ExportHandler(f.deps)(context.Background(), job)
	if err != nil {
		tb.Fatal(err)
	}
	if _, err = f.deps.Jobs.Complete(context.Background(), job.ID, "worker", result, f.now); err != nil {
		tb.Fatal(err)
	}
	var meta ports.ExportArtifact
	if err := json.Unmarshal(result, &meta); err != nil {
		tb.Fatal(err)
	}
	return meta
}

//spec:covers REQ-AUDIT-009: 要求条件とポリシーを保存し、bulk レーンで処理する。
//spec:covers REQ-JOBS-009: 種別の登録が bulk レーンを決める。
//spec:covers REQ-AUDIT-010: 同時刻の ID 境界を越えて全ページを欠落や重複なく保存する。
func TestAuditExportCompletesAllPages(t *testing.T) {
	for _, count := range []int{0, 500, 1001, 1500} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			f := newExportFixture(t, count)
			job := f.start(t, domain.DefaultExportPolicy())
			var params ports.ExportParams
			if err := json.Unmarshal(job.Params, &params); err != nil {
				t.Fatal(err)
			}
			if !params.Query.Before.Equal(f.now) || params.RequestedBy != "owner" || job.Lane != jobsdomain.LaneBulk {
				t.Fatalf("params=%+v job=%+v", params, job)
			}
			meta := f.complete(t, job)
			scope := ExportScope{TenantID: "a", RequestedBy: "owner"}
			reader, got, err := DownloadExport(context.Background(), f.deps, scope, job.ID)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			var body struct {
				Events []struct {
					ID     string `json:"id"`
					Tenant string `json:"tenant_id"`
				} `json:"events"`
			}
			if err := json.Unmarshal(data, &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Events) != count || meta.Events != count || meta.ByteSize != int64(len(data)) || got.Ref != meta.Ref {
				t.Fatalf("events=%d metadata=%+v", len(body.Events), meta)
			}
			for i, event := range body.Events {
				if event.ID != fmt.Sprintf("%08d", count-1-i) || event.Tenant != "a" {
					t.Fatalf("event[%d]=%+v", i, event)
				}
			}
			repeated, err := usecases.ExportHandler(f.deps)(context.Background(), job)
			if !errors.Is(err, jobsports.ErrJobLeaseLost) || len(repeated) != 0 {
				t.Fatalf("terminal replay err=%v", err)
			}
		})
	}
}

//spec:covers REQ-AUDIT-010: 件数とバイト数の境界のちょうどは成功し、一つ超えた成果物は公開しない。
func TestAuditExportLimitsAndFailurePublication(t *testing.T) {
	policy := domain.DefaultExportPolicy()
	for _, count := range []int{2, 3} {
		t.Run(fmt.Sprintf("events-%d", count), func(t *testing.T) {
			f := newExportFixture(t, count)
			p := policy
			p.MaxEvents = 2
			job := f.start(t, p)
			_, err := usecases.ExportHandler(f.deps)(context.Background(), job)
			if count == 2 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, usecases.ErrEventLimit) {
				t.Fatalf("err=%v", err)
			}
			assertNoExport(t, f, job.ID)
		})
	}
	f := newExportFixture(t, 2)
	var encoded bytes.Buffer
	p := ports.ExportParams{Query: ports.AuditEventQuery{TenantID: "a"}, Policy: policy}
	if _, err := usecases.StreamExport(context.Background(), f.deps, p, &encoded, nil, false); err != nil {
		t.Fatal(err)
	}
	for _, delta := range []int64{0, -1} {
		t.Run(fmt.Sprintf("bytes-%d", delta), func(t *testing.T) {
			f := newExportFixture(t, 2)
			p := policy
			p.MaxBytes = int64(encoded.Len()) + delta
			job := f.start(t, p)
			_, err := usecases.ExportHandler(f.deps)(context.Background(), job)
			if delta == 0 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, usecases.ErrByteLimit) {
				t.Fatalf("err=%v", err)
			}
			assertNoExport(t, f, job.ID)
		})
	}
}

func assertNoExport(t *testing.T, f *exportFixture, id string) {
	t.Helper()
	reader, _, err := f.deps.Artifacts.Open(context.Background(), "a", id)
	if reader != nil {
		reader.Close()
	}
	if !errors.Is(err, ports.ErrExportArtifactNotFound) {
		t.Fatalf("partial artifact exposed: %v", err)
	}
}

type interruptingEvents struct {
	ports.AuditEventRepository
	afterPage func()
	pages     int
}

func (s *interruptingEvents) List(ctx context.Context, q ports.AuditEventQuery) ([]*ports.AuditEventRecord, error) {
	if q.Limit != 500 {
		return nil, errors.New("unbounded page")
	}
	s.pages++
	if s.pages == 2 {
		s.afterPage()
	}
	return s.AuditEventRepository.List(ctx, q)
}

//spec:covers REQ-AUDIT-010: ページ途中の取消しと保存先障害は部分成果物を公開しない。
func TestAuditExportCancellationAndStorageFailure(t *testing.T) {
	f := newExportFixture(t, 1001)
	job := f.start(t, domain.DefaultExportPolicy())
	f.deps.Events = &interruptingEvents{AuditEventRepository: f.deps.Events, afterPage: func() {
		if _, err := f.deps.Jobs.Cancel(context.Background(), job.ID, f.now); err != nil {
			t.Error(err)
		}
	}}
	if _, err := usecases.ExportHandler(f.deps)(context.Background(), job); !errors.Is(err, jobsports.ErrJobLeaseLost) {
		t.Fatalf("err=%v", err)
	}
	assertNoExport(t, f, job.ID)
	f = newExportFixture(t, 1)
	job = f.start(t, domain.DefaultExportPolicy())
	f.deps.Artifacts = &failingArtifacts{ExportArtifactStore: f.deps.Artifacts}
	if _, err := usecases.ExportHandler(f.deps)(context.Background(), job); !errors.Is(err, usecases.ErrExportFailed) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("err=%v", err)
	}
	assertNoExport(t, f, job.ID)
	f = newExportFixture(t, 1)
	job = f.start(t, domain.DefaultExportPolicy())
	f.deps.Jobs = failingExportJobs{JobRepository: f.deps.Jobs}
	if _, err := usecases.ExportHandler(f.deps)(context.Background(), job); !errors.Is(err, usecases.ErrExportFailed) {
		t.Fatalf("job read failure=%v", err)
	}
}

type failingExportJobs struct{ jobsports.JobRepository }

func (failingExportJobs) Get(context.Context, string) (*jobsdomain.Job, error) {
	return nil, errors.New("secret database details")
}

type failingArtifacts struct{ ports.ExportArtifactStore }

func (*failingArtifacts) Write(context.Context, string, string, func(io.Writer) (ports.ExportCompletion, error)) (ports.ExportArtifact, error) {
	return ports.ExportArtifact{}, errors.New("secret database details")
}

//spec:covers REQ-AUDIT-011: 所有者、範囲、成功、期限を毎回照合する。
func TestAuditExportDownloadScopeAndExpiry(t *testing.T) {
	f := newExportFixture(t, 1)
	job := f.start(t, domain.DefaultExportPolicy())
	scope := ExportScope{TenantID: "a", RequestedBy: "owner"}
	if _, _, err := DownloadExport(context.Background(), f.deps, scope, job.ID); !errors.Is(err, usecases.ErrExportUnavailable) {
		t.Fatal(err)
	}
	meta := f.complete(t, job)
	for _, wrong := range []ExportScope{{TenantID: "b", RequestedBy: "owner"}, {TenantID: "a", RequestedBy: "another"}, {TenantID: "a", RequestedBy: "owner", System: true}} {
		if _, _, err := DownloadExport(context.Background(), f.deps, wrong, job.ID); !errors.Is(err, usecases.ErrExportNotFound) {
			t.Fatalf("scope=%+v err=%v", wrong, err)
		}
	}
	f.now = meta.ExpiresAt
	view, err := GetExport(context.Background(), f.deps, scope, job.ID)
	if err != nil || view.Status != "expired" || view.Downloadable {
		t.Fatalf("view=%+v err=%v", view, err)
	}
	if _, _, err := DownloadExport(context.Background(), f.deps, scope, job.ID); !errors.Is(err, usecases.ErrExportUnavailable) {
		t.Fatal(err)
	}
	if err := f.module.PurgeExpiredExports(context.Background(), f.deps.Jobs, f.now, f.deps.Emit); err != nil {
		t.Fatal(err)
	}
	usage, err := f.module.ExportUsage(context.Background(), "a")
	if err != nil || usage != 0 {
		t.Fatalf("usage=%d err=%v", usage, err)
	}
}

type pausedExportEvents struct {
	ports.AuditEventRepository
	entered chan struct{}
	release chan struct{}
}

func (p *pausedExportEvents) List(ctx context.Context, q ports.AuditEventQuery) ([]*ports.AuditEventRecord, error) {
	close(p.entered)
	select {
	case <-p.release:
		return p.AuditEventRepository.List(ctx, q)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (*pausedExportEvents) Count(context.Context, ports.AuditEventQuery) (int64, error) {
	panic("export creation must not count audit events")
}

//spec:covers REQ-AUDIT-009: Worker の読出し停止中も、作成と状態確認は監査の DB 読出しを待たない。
func TestAuditExportRequestsDoNotWaitForWorker(t *testing.T) {
	f := newExportFixture(t, 1001)
	paused := &pausedExportEvents{AuditEventRepository: f.deps.Events, entered: make(chan struct{}), release: make(chan struct{})}
	f.deps.Events = paused
	job := f.start(t, domain.DefaultExportPolicy())
	finished := make(chan error, 1)
	go func() { _, err := usecases.ExportHandler(f.deps)(context.Background(), job); finished <- err }()
	<-paused.entered
	defer close(paused.release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// イベントシンクも共有可変状態を持たせず、API と Worker の並行実行を検査する。
	deps := f.deps
	deps.Emit = func(spec.DomainEvent) {}
	deps.Enqueuer = jobs.NewEnqueuer(deps.Jobs, deps.Quotas, deps.Emit)
	for range 20 {
		view, err := StartExport(ctx, deps, ports.ExportParams{Query: ports.AuditEventQuery{TenantID: "a"}, RequestedBy: "owner", Format: "json", Policy: domain.DefaultExportPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := GetExport(ctx, deps, ExportScope{TenantID: "a", RequestedBy: "owner"}, view.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := usecases.CancelExport(ctx, deps, ExportScope{TenantID: "a", RequestedBy: "owner"}, job.ID); err != nil {
		t.Fatal(err)
	}
	// 解放後に取消しを確認して終了する。
	paused.release <- struct{}{}
	if err := <-finished; !errors.Is(err, jobsports.ErrJobLeaseLost) {
		t.Fatalf("worker err=%v", err)
	}
}

func BenchmarkAuditExportRequest(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			f := newExportFixture(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				deps := f.deps
				deps.Jobs = jobs.NewJobRepository()
				deps.Quotas = tenancy.NewMemoryModule().QuotaRepo
				deps.Emit = func(spec.DomainEvent) {}
				deps.Enqueuer = jobs.NewEnqueuer(deps.Jobs, deps.Quotas, deps.Emit)
				if _, err := StartExport(context.Background(), deps, ports.ExportParams{Query: ports.AuditEventQuery{TenantID: "a"}, RequestedBy: "owner", Format: "json", Policy: domain.DefaultExportPolicy()}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

//spec:covers REQ-AUDIT-011: 成果物容量と上限、処理段階を識別子のラベルなしに公開する。
func TestAuditExportMetrics(t *testing.T) {
	f := newExportFixture(t, 1)
	f.deps.Metrics = f.module.metrics
	job := f.start(t, domain.DefaultExportPolicy())
	meta := f.complete(t, job)
	f.module.metrics.Limit("event_limit_exceeded")
	registry := prometheus.NewRegistry()
	registry.MustRegister(f.module.ExportCollector())
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, family := range families {
		seen[family.GetName()] = true
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				if label.GetName() != "operation" && label.GetName() != "outcome" && label.GetName() != "limit" {
					t.Fatalf("unsafe label=%s", label.GetName())
				}
			}
			if family.GetName() == "idmagic_audit_export_artifact_bytes" && metric.Gauge.GetValue() != float64(meta.ByteSize) {
				t.Fatalf("bytes=%v", metric.Gauge)
			}
		}
	}
	for _, name := range []string{"idmagic_audit_export_artifact_bytes", "idmagic_audit_export_limits_total", "idmagic_audit_export_operations_total"} {
		if !seen[name] {
			t.Fatalf("missing %s", name)
		}
	}
}
