package scim

import (
	"github.com/labstack/echo/v5"

	"idmagic/internal/shared/adapters/http/support"
)

func RegisterRoutes(g *echo.Group, sd *support.Deps, u *Usecases) {
	h := NewHandler(u, *sd)

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

	// Admin API for SCIM management
	g.GET("/api/admin/scim/config", h.handleGetAdminConfig)
	g.PUT("/api/admin/scim/config", h.handleUpdateAdminConfig)
	g.GET("/api/admin/scim/tokens", h.handleListAdminTokens)
	g.POST("/api/admin/scim/tokens", h.handleCreateAdminToken)
	g.DELETE("/api/admin/scim/tokens/:id", h.handleRevokeAdminToken)
}
