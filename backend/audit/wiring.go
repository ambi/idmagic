package audit

import (
	"github.com/ambi/idmagic/backend/audit/internal/db_memory"
	"github.com/ambi/idmagic/backend/audit/internal/db_postgres"
	"github.com/ambi/idmagic/backend/audit/internal/handlers_http"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/salts_memory"
	"github.com/ambi/idmagic/backend/shared/security/salts_postgres"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"

	"github.com/labstack/echo/v5"
)

func NewAuditEventStore(maxEvents int) *db_memory.AuditEventStore {
	return db_memory.NewAuditEventStore(maxEvents)
}

func NewMemoryModule(maxEvents int) Module {
	return Module{AuditEventRepo: NewAuditEventStore(maxEvents), TenantSaltStore: salts_memory.NewInMemoryTenantSaltStore()}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{AuditEventRepo: &db_postgres.AuditEventRepository{Pool: pool}, TenantSaltStore: salts_postgres.NewTenantSaltStore(pool)}
}

// RecordWriterInTx は同じ Tx へ監査を追記し、検索の対象を明示できる書き込み操作を返す。
func RecordWriterInTx(tx pgx.Tx) ports.AuditEventWriter { return db_postgres.RecordWriterInTx(tx) }

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	LookupUsername ports.UsernameLookup
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator, LookupUsername: d.LookupUsername,
		AuditEventRepo: m.AuditEventRepo, TenantSaltStore: m.TenantSaltStore,
	})
}

func ExtractSearchAttributes(record *ports.AuditEventRecord) map[string][]string {
	return usecases.ExtractSearchAttributes(record)
}
