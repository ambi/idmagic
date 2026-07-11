// Package scim は SCIM bounded context の DI 組立を所有する (ADR-091)。
package scim

import (
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	scimhttp "github.com/ambi/idmagic/backend/scim/adapters/http"
	"github.com/ambi/idmagic/backend/scim/ports"
	scimusecases "github.com/ambi/idmagic/backend/scim/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type Module struct {
	Repo ports.ScimRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	userRepo idmports.UserRepository, groupRepo idmports.GroupRepository, emit func(spec.DomainEvent),
) {
	scimhttp.RegisterRoutes(g, scimhttp.Deps{
		Deps: deps, Authenticator: authenticator,
		Usecases: scimusecases.NewUsecases(m.Repo, userRepo, groupRepo, emit),
	})
}
