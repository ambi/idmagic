package workloadidentity

import (
	"context"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/db_memory"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/db_postgres"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/handlers_http"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/usecases"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/verification_jose"

	"github.com/labstack/echo/v5"
)

func NewMemoryModule() Module {
	return Module{TrustBundleRepo: db_memory.NewWorkloadTrustBundleRepository(), BindingRepo: db_memory.NewAgentWorkloadBindingRepository()}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{TrustBundleRepo: &db_postgres.WorkloadTrustBundleRepository{Pool: pool}, BindingRepo: &db_postgres.AgentWorkloadBindingRepository{Pool: pool}}
}

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	AgentRepo agentports.AgentRepository
	FetchJWKS func(context.Context, *domain.WorkloadTrustBundle) ([]map[string]any, error)
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator, AgentRepo: d.AgentRepo,
		TrustBundleRepo: m.TrustBundleRepo, BindingRepo: m.BindingRepo, FetchJWKS: d.FetchJWKS,
	})
}

// TokenVerifier は管理 API と同じ JWKS 供給元を使う、OAuth2 向けの検証器を組み立てる。
func (m Module) TokenVerifier(agents agentports.AgentRepository, fetch func(context.Context, *domain.WorkloadTrustBundle) ([]map[string]any, error), emit func(spec.DomainEvent)) usecases.WorkloadTokenVerifierAdapter {
	return usecases.WorkloadTokenVerifierAdapter{Deps: usecases.VerifyWorkloadAttestationDeps{
		TrustBundleRepo: m.TrustBundleRepo, BindingRepo: m.BindingRepo, AgentRepo: agents,
		SVIDVerifier: verification_jose.NewVerifier(), FetchJWKS: fetch, Emit: emit,
	}}
}
