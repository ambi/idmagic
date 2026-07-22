package handlers_http

import (
	"github.com/labstack/echo/v5"

	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	"github.com/ambi/idmagic/backend/scim/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	Usecases              *usecases.Usecases
	ApiTokenAuthenticator apitokenports.Authenticator
}

func RegisterRoutes(g *echo.Group, d Deps) {
	h := NewHandler(d)

	// SCIM 2.0 Endpoints
	g.GET("/scim/v2/ServiceProviderConfig", h.handleGetServiceProviderConfig)
	g.GET("/scim/v2/ResourceTypes", h.handleGetResourceTypes)
	g.GET("/scim/v2/Schemas", h.handleGetSchemas)

	g.GET("/scim/v2/Users", h.handleListUsers)
	g.POST("/scim/v2/Users", h.handleCreateUser)
	g.GET("/scim/v2/Users/:id", h.handleGetUser)
	g.PUT("/scim/v2/Users/:id", h.handleUpdateUser)
	g.PATCH("/scim/v2/Users/:id", h.handlePatchUser)
	g.DELETE("/scim/v2/Users/:id", h.handleDeleteUser)

	g.GET("/scim/v2/Groups", h.handleListGroups)
	g.POST("/scim/v2/Groups", h.handleCreateGroup)
	g.GET("/scim/v2/Groups/:id", h.handleGetGroup)
	g.PUT("/scim/v2/Groups/:id", h.handleUpdateGroup)
	g.PATCH("/scim/v2/Groups/:id", h.handlePatchGroup)
	g.DELETE("/scim/v2/Groups/:id", h.handleDeleteGroup)
}
