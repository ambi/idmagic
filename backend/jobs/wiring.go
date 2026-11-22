package jobs

import (
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/jobs/internal/db_memory"
	"github.com/ambi/idmagic/backend/jobs/internal/db_postgres"
	jobshttp "github.com/ambi/idmagic/backend/jobs/internal/handlers_http"
	"github.com/ambi/idmagic/backend/jobs/internal/metrics_prometheus"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/internal/usecases"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// 組み立て地点が worker を構築するときに名指す型である。
type (
	HandlerRegistry = jobsusecases.HandlerRegistry
	Runner          = jobsusecases.Runner
	RunnerConfig    = jobsusecases.RunnerConfig
	RunnerDeps      = jobsusecases.RunnerDeps
	Metrics         = metrics_prometheus.Metrics
)

func NewMemoryModule() Module { return Module{Repo: db_memory.NewJobRepository()} }

func NewPostgresModule(pool postgres.DB) Module {
	return Module{Repo: &db_postgres.JobRepository{Pool: pool}}
}

// NewJobRepository はメモリのジョブ保存先を返す。
func NewJobRepository() jobsports.JobRepository { return db_memory.NewJobRepository() }

// NewEnqueuer は、repo へジョブを作り、quotas で active_jobs を確認し、emit で JobEnqueued を発行する投入器を返す。
func NewEnqueuer(repo jobsports.JobRepository, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) jobsports.Enqueuer {
	return jobsusecases.NewEnqueuer(repo, quotas, emit)
}

// NewEnqueuerInTx は呼び出し側の Tx で、通常と同じ投入規則と active_jobs の確認を適用する。
// quotas も同じ Tx に結ぶ。emit は確定までイベントを保持し、確定後に発行する関数を渡す。
// 拒否の QuotaExceeded はロールバック後に発行し、失敗した投入の JobEnqueued は破棄する。
func NewEnqueuerInTx(tx pgx.Tx, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) jobsports.Enqueuer {
	if tx == nil {
		panic("jobs: NewEnqueuerInTx requires a transaction")
	}
	return jobsusecases.NewEnqueuer(&db_postgres.JobRepository{Pool: tx}, quotas, emit)
}

// NewCanceler は、repo でジョブを取り消し、quotas の active_jobs を一つ減らし、emit で JobCanceled を発行する取り消しの操作を返す。
func NewCanceler(repo jobsports.JobRepository, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) jobsports.Canceler {
	return jobsusecases.NewCanceler(repo, quotas, emit)
}

func NewHandlerRegistry() *HandlerRegistry { return jobsusecases.NewHandlerRegistry() }

func NewRunner(cfg RunnerConfig, deps RunnerDeps) *Runner { return jobsusecases.NewRunner(cfg, deps) }

// NewMetrics は Jobs の計測と、同じ provider による HTTP 計測と公開を作る。
func NewMetrics(serviceName, serviceVersion string) (*Metrics, error) {
	return metrics_prometheus.NewMetrics(serviceName, serviceVersion)
}

// RegisterRoutes は、テナント管理とシステムのジョブの参照と取り消しの経路を登録する。
// 取り消しは m.Canceler を通し、active_jobs の減算と JobCanceled の発行をそこに任せる。
func (m Module) RegisterRoutes(g *echo.Group, deps support.Deps, authenticator *support.Authenticator) {
	jobshttp.RegisterRoutes(g, jobshttp.Deps{Deps: deps, Authenticator: authenticator, Repo: m.Repo, Canceler: m.Canceler})
}
