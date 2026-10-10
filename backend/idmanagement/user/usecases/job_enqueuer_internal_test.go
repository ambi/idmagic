package usecases

import (
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
)

// testJobEnqueuer は、repo へジョブを作り、上限のないクォータで数え、イベントを捨てる投入器を返す。
func testJobEnqueuer(repo jobsports.JobRepository) jobsports.Enqueuer {
	return jobsusecases.NewEnqueuer(repo, tenancymemory.NewQuotaRepository(), func(spec.DomainEvent) {})
}
