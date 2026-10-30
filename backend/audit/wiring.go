package audit

import (
	"context"
	"crypto/rand"

	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/internal/db_memory"
	"github.com/ambi/idmagic/backend/audit/internal/db_postgres"
	"github.com/ambi/idmagic/backend/audit/internal/handlers_http"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/salts_memory"
	"github.com/ambi/idmagic/backend/shared/security/salts_postgres"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"

	"github.com/labstack/echo/v5"
)

func NewAuditEventStore(maxEvents int) *db_memory.AuditEventStore {
	return db_memory.NewAuditEventStore(maxEvents)
}

func withExportMetrics(m Module) Module { m.metrics = newExportMetrics(m.ExportArtifacts); return m }

func NewMemoryModule(maxEvents int) Module {
	return withExportMetrics(Module{AuditEventRepo: NewAuditEventStore(maxEvents), TenantSaltStore: salts_memory.NewInMemoryTenantSaltStore(), ExportArtifacts: db_memory.NewExportArtifactStore(), ExportPolicy: domain.DefaultExportPolicy(), ExportCodec: support.NewCursorCodec([]byte(rand.Text()))})
}

func NewPostgresModule(pool postgres.DB) Module {
	return withExportMetrics(Module{AuditEventRepo: &db_postgres.AuditEventRepository{Pool: pool}, TenantSaltStore: salts_postgres.NewTenantSaltStore(pool), ExportArtifacts: &db_postgres.ExportArtifactStore{Pool: pool}, ExportPolicy: domain.DefaultExportPolicy(), ExportCodec: support.NewCursorCodec([]byte(rand.Text()))})
}

// RecordWriterInTx は同じ Tx へ監査を追記し、検索の対象を明示できる書き込み操作を返す。
func RecordWriterInTx(tx pgx.Tx) ports.AuditEventWriter { return db_postgres.RecordWriterInTx(tx) }

// CaptureInTx は監査の記録と同時に確定する作用である。失敗は追記を巻き戻す。
type CaptureInTx = db_postgres.CaptureInTx

// RecordWriterInTxCapturing は RecordWriterInTx に、記録と同時に確定する作用を加える。
func RecordWriterInTxCapturing(capture CaptureInTx) func(pgx.Tx) ports.AuditEventWriter {
	return func(tx pgx.Tx) ports.AuditEventWriter { return db_postgres.RecordWriterInTxCapturing(tx, capture) }
}

// WithCapture は記録と同時に確定する作用を結ぶ。Audit は呼び先を知らず、組み立て地点が渡す。
// PostgreSQL は inTx を同じ Tx で、メモリは inMemory を追記の中で呼ぶ。
func (m Module) WithCapture(inTx CaptureInTx, inMemory func(context.Context, *ports.AuditEventRecord) error) Module {
	switch repo := m.AuditEventRepo.(type) {
	case *db_postgres.AuditEventRepository:
		repo.CaptureInTx = inTx
	case *db_memory.AuditEventStore:
		repo.SetCapture(inMemory)
	}
	return m
}

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	LookupUsername ports.UsernameLookup
	Jobs           jobsports.JobRepository
	Enqueuer       jobsports.Enqueuer
	Quotas         tenantports.QuotaRepository
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	codec := m.ExportCodec
	if codec == nil {
		codec = support.NewCursorCodec([]byte(rand.Text()))
	}
	policy := m.ExportPolicy
	if policy.MaxEvents == 0 {
		policy = domain.DefaultExportPolicy()
	}
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator, LookupUsername: d.LookupUsername,
		AuditEventRepo: m.AuditEventRepo, TenantSaltStore: m.TenantSaltStore,
		Exports: usecases.ExportDeps{Events: m.AuditEventRepo, Artifacts: m.ExportArtifacts, Jobs: d.Jobs, Enqueuer: d.Enqueuer, Quotas: d.Quotas, Emit: d.Emit, Codec: codec, Metrics: m.metrics}, ExportPolicy: policy,
	})
}

func ExtractSearchAttributes(record *ports.AuditEventRecord) map[string][]string {
	return usecases.ExtractSearchAttributes(record)
}
