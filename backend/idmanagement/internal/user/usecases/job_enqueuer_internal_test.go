package usecases

import (
	jobsmodule "github.com/ambi/idmagic/backend/jobs"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// testJobEnqueuer は、repo へジョブを作り、上限のないクォータで数え、イベントを捨てる投入器を返す。
func testJobEnqueuer(repo jobsports.JobRepository) jobsports.Enqueuer {
	return jobsmodule.NewEnqueuer(repo, tenancy.NewMemoryModule().QuotaRepo, func(spec.DomainEvent) {})
}
