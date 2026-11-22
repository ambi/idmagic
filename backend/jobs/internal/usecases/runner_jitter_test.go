package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/internal/usecases"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-JOBS-005: 正式な Runner の構築経路で注入した待ち時間を run_at と JobRetried に保存する。
func TestRunnerRetryUsesRandomDelay(t *testing.T) {
	for _, delay := range []time.Duration{0, 7 * time.Second, 30 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				now := time.Now().UTC()
				repo := jobs.NewJobRepository()
				job, err := jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: repo}, jobsports.EnqueueInput{
					TenantID: "tenant-a", Kind: jobsdomain.KindNoopEcho, MaxAttempts: 3,
				}, now)
				if err != nil {
					t.Fatal(err)
				}
				handlers := jobs.NewHandlerRegistry()
				handlers.Register(jobsdomain.KindNoopEcho, func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
					return nil, errors.New("unavailable")
				})
				var limit time.Duration
				var retriedAt time.Time
				runner := jobs.NewRunner(jobs.RunnerConfig{
					WorkerID: "jitter-worker", Lane: jobsdomain.LaneDefault, PollInterval: time.Second, Concurrency: 1,
				}, jobs.RunnerDeps{
					Repo: repo, Handlers: handlers, Now: func() time.Time { return now },
					RandomDelay: func(upper time.Duration) time.Duration { limit = upper; return delay },
					Emit: func(e spec.DomainEvent) {
						if retried, ok := e.(*jobsdomain.JobRetried); ok {
							retriedAt = retried.NextRunAt
							cancel()
						}
					},
				})
				if err := runner.Run(ctx); !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				got, err := repo.Get(context.Background(), job.ID)
				if err != nil {
					t.Fatal(err)
				}
				if limit != 30*time.Second || got.Status != jobsdomain.StatusQueued || got.Attempts != 1 ||
					!got.RunAt.Equal(now.Add(delay)) || !retriedAt.Equal(got.RunAt) {
					t.Fatalf("limit=%v job=%+v event.run_at=%v, want queued at %v", limit, got, retriedAt, now.Add(delay))
				}
			})
		})
	}
}
