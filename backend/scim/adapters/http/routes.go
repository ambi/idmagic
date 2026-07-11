package http

import (
	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/scim/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	Usecases *usecases.Usecases
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

	// Admin API for SCIM access tokens
	g.GET("/api/admin/scim/tokens", h.handleListAdminTokens)
	g.POST("/api/admin/scim/tokens", h.handleCreateAdminToken)
	g.DELETE("/api/admin/scim/tokens/:id", h.handleRevokeAdminToken)
}
