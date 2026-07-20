// Package scim は SCIM bounded context の DI 組立を所有する (ADR-091)。
package scim

import (
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	scimhttp "github.com/ambi/idmagic/backend/scim/handlers_http"
	"github.com/ambi/idmagic/backend/scim/ports"
	scimusecases "github.com/ambi/idmagic/backend/scim/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type Module struct {
	Repo ports.ScimRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	userRepo userports.UserRepository, groupRepo groupports.GroupRepository, emit func(spec.DomainEvent),
) {
	scimhttp.RegisterRoutes(g, scimhttp.Deps{
		Deps: deps, Authenticator: authenticator,
		Usecases: scimusecases.NewUsecases(m.Repo, userRepo, groupRepo, emit),
	})
}
