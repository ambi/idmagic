// Package sourcing は Sourcing bounded context の DI 組立を所有する。
// source slice は現在 scim (SCIM 2.0 server) のみ。source 非依存コアは 2 つ目の source が
// 着地した時点で on-demand に切り出す (決定 3・4)。
package sourcing

import (
	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	scimhttp "github.com/ambi/idmagic/backend/sourcing/scim/handlers_http"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
	scimusecases "github.com/ambi/idmagic/backend/sourcing/scim/usecases"

	"github.com/labstack/echo/v5"
)

type Module struct {
	ScimRepo ports.ScimRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	userRepo userports.UserRepository, groupRepo groupports.GroupRepository, emit func(spec.DomainEvent),
	apiTokenAuthenticator apitokenports.Authenticator,
) {
	scimhttp.RegisterRoutes(g, scimhttp.Deps{
		Deps: deps, Authenticator: authenticator,
		Usecases:              scimusecases.NewUsecases(m.ScimRepo, userRepo, groupRepo, emit),
		ApiTokenAuthenticator: apiTokenAuthenticator,
	})
}
