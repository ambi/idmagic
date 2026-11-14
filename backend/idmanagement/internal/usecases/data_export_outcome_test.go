package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type controlledCSVExporter struct {
	idmusecases.UserCSVExporter
	generate func(context.Context) (idmports.CSVArtifact, int, error)
}

func (e controlledCSVExporter) ExportUserCSV(ctx context.Context, _ []string, _ string) (idmports.CSVArtifact, int, error) {
	return e.generate(ctx)
}

type unsettledExportRepository struct {
	jobsports.JobRepository
	err error
}

func (r unsettledExportRepository) Complete(context.Context, string, string, json.RawMessage, time.Time) (*jobsdomain.Job, error) {
	return nil, r.err
}

func (r unsettledExportRepository) Fail(context.Context, string, string, jobsports.FailOutcome, time.Time) (*jobsdomain.Job, error) {
	return nil, r.err
}

//spec:covers REQ-IDMANAGEMENT-086: 成功と最終失敗の保存後だけ結果イベントを発行し、再試行、取り消し、リース喪失、保存失敗からは発行しない。
func TestDataExportOutcomeEventsFollowCommittedStatus(t *testing.T) {
	limitError := &idmdomain.CSVError{Code: idmdomain.CSVErrorTooManyRows}
	for _, tc := range []struct {
		name        string
		failure     error
		maxAttempts int
		interrupt   string
		commitError error
		wantStatus  jobsdomain.JobStatus
		wantEvents  []string
		wantError   string
	}{
		{name: "success", maxAttempts: 1, wantStatus: jobsdomain.StatusSucceeded, wantEvents: []string{"DataExportSucceeded"}},
		{name: "final failure", failure: limitError, maxAttempts: 1, wantStatus: jobsdomain.StatusFailed, wantEvents: []string{"DataExportFailed"}, wantError: "csv_transfer_limit_exceeded"},
		{name: "final ordinary failure", failure: errors.New("生成器を利用できない"), maxAttempts: 1, wantStatus: jobsdomain.StatusFailed, wantEvents: []string{"DataExportFailed"}, wantError: "export_failed"},
		{name: "retry", failure: limitError, maxAttempts: 2, wantStatus: jobsdomain.StatusQueued},
		{name: "canceled success", maxAttempts: 1, interrupt: "cancel", wantStatus: jobsdomain.StatusCanceled},
		{name: "canceled failure", failure: limitError, maxAttempts: 1, interrupt: "cancel", wantStatus: jobsdomain.StatusCanceled},
		{name: "reclaimed success", maxAttempts: 2, interrupt: "reclaim", wantStatus: jobsdomain.StatusRunning},
		{name: "reclaimed failure", failure: limitError, maxAttempts: 2, interrupt: "reclaim", wantStatus: jobsdomain.StatusRunning},
		{name: "complete storage failure", maxAttempts: 1, commitError: errors.New("保存先を利用できない"), wantStatus: jobsdomain.StatusRunning},
		{name: "fail storage failure", failure: limitError, maxAttempts: 1, commitError: errors.New("保存先を利用できない"), wantStatus: jobsdomain.StatusRunning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, _ := seededExportDeps(t)
			ctx := exportTestCtx()
			now := deps.Now()
			job, err := deps.Enqueuer.Enqueue(ctx, jobsports.EnqueueInput{
				TenantID: "acme", Kind: idmusecases.KindDataExport, MaxAttempts: tc.maxAttempts,
				Params: mustMarshal(t, idmusecases.DataExportParams{Target: "user", Columns: []string{"preferred_username"}}),
			}, now)
			if err != nil {
				t.Fatal(err)
			}
			runCtx, stop := context.WithCancel(ctx)
			defer stop()
			deps.UserCSVExporter = controlledCSVExporter{generate: func(ctx context.Context) (idmports.CSVArtifact, int, error) {
				// 停止は新たな取得だけを止める。Runner はこの試行の確定と通知を終えてから戻る。
				stop()
				switch tc.interrupt {
				case "cancel":
					if _, err := deps.JobCanceler.Cancel(ctx, job.ID, now); err != nil {
						return idmports.CSVArtifact{}, 0, err
					}
				case "reclaim":
					if _, err := deps.JobRepo.ClaimBatch(ctx, "other-worker", jobsdomain.LaneBulk, 1, time.Hour, now.Add(2*time.Hour)); err != nil {
						return idmports.CSVArtifact{}, 0, err
					}
				}
				if tc.failure != nil {
					return idmports.CSVArtifact{}, 0, tc.failure
				}
				return idmports.CSVArtifact{Ref: "artifact", SHA256: "digest", ByteSize: 64}, 2, nil
			}}
			var outcomes []spec.DomainEvent
			var jobEvents []string
			started := 0
			deps.Emit = func(event spec.DomainEvent) error {
				if event.EventType() == "DataExportStarted" {
					started++
					return nil
				}
				stored, err := deps.JobRepo.Get(ctx, job.ID)
				if err != nil {
					return err
				}
				if stored.Status != tc.wantStatus || stored.Status == jobsdomain.StatusRunning {
					t.Errorf("結果イベント %s の発行時に status=%s", event.EventType(), stored.Status)
				}
				outcomes = append(outcomes, event)
				return nil
			}
			repo := deps.JobRepo
			if tc.commitError != nil {
				repo = unsettledExportRepository{JobRepository: repo, err: tc.commitError}
			}
			registry := jobs.NewHandlerRegistry()
			registry.Register(idmusecases.KindDataExport, idmusecases.DataExportHandler(deps), idmusecases.DataExportSettlementHandler(deps))
			runner := jobs.NewRunner(jobs.RunnerConfig{
				WorkerID: "export-worker", Lane: jobsdomain.LaneBulk, Concurrency: 1,
				PollInterval: time.Millisecond, LeaseDuration: time.Hour,
			}, jobs.RunnerDeps{Repo: repo, Handlers: registry, Now: deps.Now, Emit: func(event spec.DomainEvent) {
				jobEvents = append(jobEvents, event.EventType())
			}})
			done := make(chan error, 1)
			go func() { done <- runner.Run(runCtx) }()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Runner が終了しない")
			}
			stored, err := deps.JobRepo.Get(ctx, job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Status != tc.wantStatus || started != 1 {
				t.Fatalf("status=%s, started=%d", stored.Status, started)
			}
			var types []string
			for _, event := range outcomes {
				types = append(types, event.EventType())
				switch e := event.(type) {
				case *idmdomain.DataExportSucceeded:
					if e.ExportID != job.ID || e.TenantID != "acme" || e.Target != "user" || e.TotalRows != 2 || e.ByteSize != 64 || !e.At.Equal(stored.UpdatedAt) {
						t.Errorf("成功イベント=%+v", e)
					}
				case *idmdomain.DataExportFailed:
					if e.ExportID != job.ID || e.TenantID != "acme" || e.Target != "user" || e.ErrorCode != tc.wantError || stored.Error == nil || *stored.Error != tc.wantError || !e.At.Equal(stored.UpdatedAt) {
						t.Errorf("失敗イベント=%+v", e)
					}
				}
			}
			if !slices.Equal(types, tc.wantEvents) {
				t.Errorf("結果イベント=%v, want %v", types, tc.wantEvents)
			}
			if tc.name == "retry" && (!slices.Contains(jobEvents, "JobRetried") || !slices.Contains(jobEvents, "JobFailed")) {
				t.Errorf("再試行の記録=%v", jobEvents)
			}
		})
	}
}
