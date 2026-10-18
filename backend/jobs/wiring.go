package jobs

import (
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/jobs/internal/db_memory"
	"github.com/ambi/idmagic/backend/jobs/internal/db_postgres"
	jobshttp "github.com/ambi/idmagic/backend/jobs/internal/handlers_http"
	"github.com/ambi/idmagic/backend/jobs/internal/metrics_prometheus"
	"github.com/ambi/idmagic/backend/jobs/internal/usecases"
	"github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// 組み立て地点が worker を構築するときに名指す型である。
type (
	HandlerRegistry = usecases.HandlerRegistry
	Runner          = usecases.Runner
	RunnerConfig    = usecases.RunnerConfig
	RunnerDeps      = usecases.RunnerDeps
	Metrics         = metrics_prometheus.Metrics
)

func NewMemoryModule() Module { return Module{Repo: db_memory.NewJobRepository()} }

func NewPostgresModule(pool postgres.DB) Module {
	return Module{Repo: &db_postgres.JobRepository{Pool: pool}}
}

// NewJobRepository はメモリのジョブ保存先を返す。
func NewJobRepository() ports.JobRepository { return db_memory.NewJobRepository() }

// NewEnqueuer は、repo へジョブを作り、quotas で active_jobs を確認し、emit で JobEnqueued を発行する投入器を返す。
func NewEnqueuer(repo ports.JobRepository, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) ports.Enqueuer {
	return usecases.NewEnqueuer(repo, quotas, emit)
}

// NewEnqueuerInTx は呼び出し側の Tx で、通常と同じ投入規則と active_jobs の確認を適用する。
// quotas も同じ Tx に結ぶ。emit は確定までイベントを保持し、確定後に発行する関数を渡す。
// 拒否の QuotaExceeded はロールバック後に発行し、失敗した投入の JobEnqueued は破棄する。
func NewEnqueuerInTx(tx pgx.Tx, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) ports.Enqueuer {
	if tx == nil {
		panic("jobs: NewEnqueuerInTx requires a transaction")
	}
	return usecases.NewEnqueuer(&db_postgres.JobRepository{Pool: tx}, quotas, emit)
}

func NewHandlerRegistry() *HandlerRegistry { return usecases.NewHandlerRegistry() }

func NewRunner(cfg RunnerConfig, deps RunnerDeps) *Runner { return usecases.NewRunner(cfg, deps) }

// NewMetrics は Jobs の計測と、同じ provider による HTTP 計測と公開を作る。
func NewMetrics(serviceName, serviceVersion string) (*Metrics, error) {
	return metrics_prometheus.NewMetrics(serviceName, serviceVersion)
}

// RegisterRoutes は、テナント管理とシステムのジョブの参照と取り消しの経路を登録する。
func (m Module) RegisterRoutes(g *echo.Group, deps support.Deps, authenticator *support.Authenticator, emit func(spec.DomainEvent)) {
	jobshttp.RegisterRoutes(g, jobshttp.Deps{Deps: deps, Authenticator: authenticator, Repo: m.Repo, Emit: emit})
}
