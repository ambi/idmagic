// Package http is the Provisioning bounded context's admin HTTP adapter
// (wi-45 T007a, spec/contexts/provisioning.yaml interfaces). It lives in the
// protocol-agnostic core (ADR-128 decision 2), not a protocol feature slice.
// The account-facing/UI consumer is deferred to a follow-up (wi-45 T007b);
// this pass focuses on making the admin API match the SCL bindings exactly
// (routes_contract_test.go's assembled-router-vs-OpenAPI invariant).
package handlers_http

import (
	"net/http"

	appports "github.com/ambi/idmagic/backend/application/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// Deps are the Provisioning admin HTTP handlers' dependencies.
type Deps struct {
	support.Deps
	*support.Authenticator

	ConnectionRepo  ports.ProvisioningConnectionRepository
	DeliveryRepo    ports.ProvisioningDeliveryRepository
	AssignmentRepo  appports.AssignmentRepository
	UserRepo        userports.UserRepository
	NewTargetClient func(conn *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error)
}

func (d Deps) adminDeps() usecases.AdminDeps {
	return usecases.AdminDeps{
		ConnectionRepo: d.ConnectionRepo, DeliveryRepo: d.DeliveryRepo,
		AssignmentRepo: d.AssignmentRepo, UserRepo: d.UserRepo,
		NewTargetClient: d.NewTargetClient,
	}
}

// RegisterRoutes registers the Application-detail "provisioning" subroute and
// the tenant-wide read-only aggregate view
// (spec/contexts/provisioning.yaml §設定の置き場所).
func RegisterRoutes(g *echo.Group, d Deps) {
	g.POST("/api/admin/applications/:application_id/provisioning", d.handleRegisterConnection)
	g.GET("/api/admin/applications/:application_id/provisioning", d.handleGetConnection)
	g.PATCH("/api/admin/applications/:application_id/provisioning", d.handleUpdateConnection)
	g.DELETE("/api/admin/applications/:application_id/provisioning", d.handleDeleteConnection)
	g.POST("/api/admin/applications/:application_id/provisioning/test", d.handleTestConnection)
	g.POST("/api/admin/applications/:application_id/provisioning/on-demand", d.handleProvisionOnDemand)
	g.POST("/api/admin/applications/:application_id/provisioning/full-resync", d.handleStartFullResync)
	g.POST("/api/admin/applications/:application_id/provisioning/resume", d.handleResumeConnection)
	g.GET("/api/admin/applications/:application_id/provisioning/deliveries", d.handleListDeliveries)
	g.GET("/api/admin/applications/:application_id/provisioning/deliveries/:delivery_id", d.handleGetDelivery)
	g.POST("/api/admin/applications/:application_id/provisioning/deliveries/:delivery_id/retry", d.handleRetryDelivery)
	g.GET("/api/admin/provisioning/connections", d.handleListTenantConnections)
}

func (d Deps) writeError(c *echo.Context, err error) error {
	switch {
	case isNotFound(err):
		return support.WriteBrowserError(c, http.StatusNotFound, "provisioning_not_found", err.Error())
	case isConflict(err):
		return support.WriteBrowserError(c, http.StatusConflict, "provisioning_conflict", err.Error())
	default:
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
