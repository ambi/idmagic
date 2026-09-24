package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Enqueuer submits a pending ProvisioningTask as a durable Jobs.Job
// (spec/contexts/provisioning.yaml §配送・信頼性, kind provisioning_task).
// dedupKey is the task's idempotency key, so a duplicate dispatch of the
// same task is a no-op at the Jobs layer too.
type Enqueuer interface {
	EnqueueProvisioningTask(ctx context.Context, tenantID, dedupKey, taskID string) (jobID string, err error)
}

// DispatcherDeps are DispatchPendingTasks's dependencies.
type DispatcherDeps struct {
	TaskRepo ports.ProvisioningTaskRepository
	Enqueuer Enqueuer
	// Emit は関連付けに成功したプロビジョニングタスクの ProvisioningTaskStarted を発行する。
	Emit func(spec.DomainEvent)
}

// DispatchPendingTasks associates up to limit pending, unattached
// tasks with a Jobs.Job (LifecycleWorkflowRunLifecycle's dispatcher
// precedent: recovers from an API-process enqueue failure via periodic
// re-scan). It is safe to call repeatedly and from multiple worker processes:
// AttachJob only succeeds once per task (job_id IS NULL guard), so only the
// worker that attached emits ProvisioningTaskStarted.
//
// 関連付けの前に、now の時点で期限に達した ScheduledDeprovision を pending の delete
// プロビジョニングタスクへ変える。猶予期間つきの削除も、ほかのプロビジョニングタスクと同じ経路で下流へ届く。
func DispatchPendingTasks(ctx context.Context, deps DispatcherDeps, limit int, now time.Time) (dispatched int, err error) {
	if err := materializeDueDeprovisions(ctx, deps.TaskRepo, limit, now); err != nil {
		return 0, err
	}
	tasks, err := deps.TaskRepo.ListUnenqueued(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, d := range tasks {
		jobID, err := deps.Enqueuer.EnqueueProvisioningTask(ctx, d.TenantID, d.IdempotencyKey(), d.ID)
		if err != nil {
			return dispatched, err
		}
		attached, err := deps.TaskRepo.AttachJob(ctx, d.TenantID, d.ID, jobID)
		if err != nil {
			return dispatched, err
		}
		if attached {
			dispatched++
			emit(deps.Emit, &domain.ProvisioningTaskStarted{
				At: now.UTC(), TenantID: d.TenantID, ConnectionID: d.ConnectionID, TaskID: d.ID, JobID: jobID,
			})
		}
	}
	return dispatched, nil
}

// materializeDueDeprovisions は期限に達した予約をプロビジョニングタスクへ変える。取消と競合した予約は
// MaterializeDeprovision が false を返すので、プロビジョニングタスクを作らずに読み飛ばす。
func materializeDueDeprovisions(ctx context.Context, repo ports.ProvisioningTaskRepository, limit int, now time.Time) error {
	due, err := repo.ListDueDeprovisions(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, reservation := range due {
		id, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		if _, err := repo.MaterializeDeprovision(ctx, reservation, reservation.Task(id, now)); err != nil {
			return err
		}
	}
	return nil
}
