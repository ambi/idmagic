package audit

import (
	"github.com/ambi/idmagic/backend/audit/internal/db_memory"
	"github.com/ambi/idmagic/backend/audit/internal/db_postgres"
	"github.com/ambi/idmagic/backend/audit/internal/handlers_http"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/salts_memory"
	"github.com/ambi/idmagic/backend/shared/security/salts_postgres"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

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

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	UserRepo userports.UserRepository
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator, UserRepo: d.UserRepo,
		AuditEventRepo: m.AuditEventRepo, TenantSaltStore: m.TenantSaltStore,
	})
}

func ExtractSearchAttributes(record *ports.AuditEventRecord) map[string][]string {
	return usecases.ExtractSearchAttributes(record)
}
