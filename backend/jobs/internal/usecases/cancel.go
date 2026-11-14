package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// canceler は ports.Canceler を実装する。取り消しの遷移と、それに伴う作用を一か所に置く。
type canceler struct {
	repo   ports.JobRepository
	quotas tenantports.QuotaRepository
	emit   func(spec.DomainEvent)
}

// NewCanceler は、repo でジョブを取り消し、quotas の active_jobs を一つ減らし、emit で
// JobCanceled を発行する取り消しの操作を返す。どれかを欠くと取り消しの経路ごとに作用が
// 食い違うので、nil は組み立ての誤りとして panic する。
func NewCanceler(repo ports.JobRepository, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) ports.Canceler {
	if repo == nil || quotas == nil || emit == nil {
		panic("jobs: NewCanceler requires a job repository, a quota repository, and an event sink")
	}
	return canceler{repo: repo, quotas: quotas, emit: emit}
}

// Cancel は終端に達していないジョブを取り消す。拒否したときは作用を起こさない。
// 同じジョブの取り消しと完了が競合しても、保存先の条件付きの遷移に勝った一方だけが
// 枠を戻すので、使用量を二度減らさない。
func (c canceler) Cancel(ctx context.Context, jobID string, now time.Time) (*domain.Job, error) {
	canceled, err := c.repo.Cancel(ctx, jobID, now)
	if err != nil {
		return nil, err
	}
	decrementActiveJobsQuota(ctx, c.quotas, canceled.TenantID)
	c.emit(&domain.JobCanceled{At: now, JobID: canceled.ID, TenantID: canceled.TenantID})
	return canceled, nil
}
