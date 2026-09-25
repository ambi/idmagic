package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// settleFullResync は resync の対象を数え直し、全件が終端なら完了させて FullResyncCompleted を発行する。
// 完了の書き込みは running のときだけ成立するので、同時に判定した呼び出しや再実行があっても発行は一度だけである。
func settleFullResync(ctx context.Context, repo ports.ProvisioningTaskRepository, emitFn func(spec.DomainEvent), resync *domain.FullResync, now time.Time) error {
	if resync == nil || resync.Status != domain.FullResyncStatusRunning {
		return nil
	}
	tally, err := repo.TallyFullResync(ctx, resync.TenantID, resync.ID)
	if err != nil {
		return err
	}
	completed, ok := resync.Settle(tally, now)
	if !ok {
		return nil
	}
	won, err := repo.CompleteFullResync(ctx, &completed)
	if err != nil || !won {
		return err
	}
	emit(emitFn, completed.CompletedEvent())
	return nil
}

// settleFullResyncOfTask は終端になったプロビジョニングタスクが Full Resync に属していれば、その完了を判定する。
func settleFullResyncOfTask(ctx context.Context, deps JobHandlerDeps, tenantID, taskID string, now time.Time) error {
	resync, err := deps.TaskRepo.FindFullResyncByTask(ctx, tenantID, taskID)
	if err != nil {
		return err
	}
	return settleFullResync(ctx, deps.TaskRepo, deps.Emit, resync, now)
}
