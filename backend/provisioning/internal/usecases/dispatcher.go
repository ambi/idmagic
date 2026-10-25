package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Enqueuer は pending のタスクを永続化するジョブとして投入する。
// dedupKey はタスクの冪等キーを基にし、保留後の再投入では前のジョブを区別する。
type Enqueuer interface {
	EnqueueProvisioningTask(ctx context.Context, tenantID, dedupKey, taskID string) (jobID string, err error)
}

// DispatcherDeps are DispatchPendingTasks's dependencies.
type DispatcherDeps struct {
	TaskRepo       ports.ProvisioningTaskRepository
	ConnectionRepo ports.ProvisioningConnectionRepository
	Enqueuer       Enqueuer
	// Emit は関連付けに成功したプロビジョニングタスクの ProvisioningTaskStarted を発行する。
	Emit func(spec.DomainEvent)
}

// DispatchPendingTasks は隔離されていない接続の pending を上限までジョブへ関連付ける。
// 関連付けは pending と走査時の job_id の一致を条件にし、成功した worker だけが
// ProvisioningTaskStarted を発行する。隔離中のタスクはページを進めて読み飛ばす。
//
// 関連付けの前に、now の時点で期限に達した ScheduledDeprovision を pending の delete
// プロビジョニングタスクへ変える。猶予期間つきの削除も、ほかのプロビジョニングタスクと同じ経路で下流へ届く。
func DispatchPendingTasks(ctx context.Context, deps DispatcherDeps, limit int, now time.Time) (dispatched int, err error) {
	if err := materializeDueDeprovisions(ctx, deps.TaskRepo, limit, now); err != nil {
		return 0, err
	}
	var afterCreatedAt time.Time
	var afterID string
	for dispatched < limit {
		tasks, err := deps.TaskRepo.ListUnenqueued(ctx, afterCreatedAt, afterID, limit-dispatched)
		if err != nil {
			return dispatched, err
		}
		if len(tasks) == 0 {
			break
		}
		for _, d := range tasks {
			afterCreatedAt, afterID = d.CreatedAt, d.ID
			conn, err := deps.ConnectionRepo.Find(ctx, d.TenantID, d.ConnectionID)
			if err != nil {
				return dispatched, err
			}
			if conn == nil || conn.Health == domain.HealthQuarantined {
				continue
			}
			dedupKey := d.IdempotencyKey()
			if d.JobID != nil {
				dedupKey += ":" + *d.JobID
			}
			jobID, err := deps.Enqueuer.EnqueueProvisioningTask(ctx, d.TenantID, dedupKey, d.ID)
			if err != nil {
				return dispatched, err
			}
			attached, err := deps.TaskRepo.AttachJob(ctx, d.TenantID, d.ID, jobID, d.JobID)
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
