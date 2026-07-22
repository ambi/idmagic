// Package apitoken owns tenant-scoped API access token composition.
package apitoken

import (
	"github.com/labstack/echo/v5"

	apitokenhttp "github.com/ambi/idmagic/backend/apitoken/handlers_http"
	"github.com/ambi/idmagic/backend/apitoken/ports"
	"github.com/ambi/idmagic/backend/apitoken/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

type Module struct {
	Repo ports.Repository
}

func (m Module) Service() *usecases.Service {
	return usecases.New(m.Repo)
}

func (m Module) Register(group *echo.Group, deps support.Deps, authenticator *support.Authenticator) {
	apitokenhttp.RegisterRoutes(group, apitokenhttp.Deps{
		Deps: deps, Authenticator: authenticator, Service: m.Service(),
	})
}
