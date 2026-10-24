// Package handlers_http は Directory Connector の管理 API と Connector API を HTTP に写す。
package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	Service   *usecases.Service
	Ingestion *usecases.Ingestion
}

func RegisterRoutes(g *echo.Group, d Deps) {
	h := handler{deps: d}
	g.POST("/api/admin/v1/directory-connectors", h.createConnector)
	g.GET("/api/admin/v1/directory-connectors", h.listConnectors)
	g.GET("/api/admin/v1/directory-connectors/:connector_id", h.getConnector)
	g.POST("/api/admin/v1/directory-connectors/:connector_id/enrollment-tokens", h.reissueEnrollment)
	g.POST("/api/admin/v1/directory-connectors/:connector_id/disable", h.disableConnector)
	g.POST("/directory-connector/v1/enrollment", h.enrollConnector)
	g.POST("/directory-connector/v1/heartbeat", h.reportHeartbeat)
	g.GET("/api/admin/v1/directory-connectors/:connector_id/ingestion-runs", h.listIngestionRuns)
	g.GET("/api/admin/v1/directory-connectors/:connector_id/ingestion-runs/:run_id", h.getIngestionRun)
	g.POST("/directory-connector/v1/ingestion-runs", h.startIngestionRun)
	g.POST("/directory-connector/v1/ingestion-runs/:run_id/user-batches", h.sendUserBatch)
	g.POST("/directory-connector/v1/ingestion-runs/:run_id/complete", h.completeIngestionRun)
}

type handler struct{ deps Deps }

type connectorResponse struct {
	ID                  string     `json:"id"`
	DisplayName         string     `json:"display_name"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	EnrollmentExpiresAt *time.Time `json:"enrollment_expires_at,omitempty"`
	EnrolledAt          *time.Time `json:"enrolled_at,omitempty"`
	KeyID               string     `json:"key_id,omitempty"`
	LastSeenAt          *time.Time `json:"last_seen_at,omitempty"`
	DisabledAt          *time.Time `json:"disabled_at,omitempty"`
}

// connectorJSON は登録トークンのハッシュと公開鍵を含めずに Connector を写す。
// 鍵の ID は Active の間だけ、登録トークンの有効期限は登録待ちの間だけ返す。
func connectorJSON(c domain.Connector) connectorResponse {
	response := connectorResponse{
		ID: c.ID, DisplayName: c.DisplayName, Status: string(c.Status), CreatedAt: c.CreatedAt,
		EnrolledAt: c.EnrolledAt, LastSeenAt: c.LastSeenAt, DisabledAt: c.DisabledAt,
	}
	switch c.Status {
	case domain.StatusActive:
		response.KeyID = c.KeyID
	case domain.StatusAwaitingEnrollment:
		expires := c.EnrollmentExpiresAt
		response.EnrollmentExpiresAt = &expires
	}
	return response
}

func grantJSON(grant usecases.Grant) map[string]any {
	return map[string]any{"connector": connectorJSON(grant.Connector), "enrollment_token": grant.Token}
}

// requireAdministrator は操作を admin と system_admin に開く。
func (h handler) requireAdministrator(c *echo.Context) error {
	if _, err := h.deps.RequireAdministrator(c); err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}
	return nil
}

// requireAdministratorChange は、変更の操作でブラウザーの要求の検査を先に行う。
func (h handler) requireAdministratorChange(c *echo.Context) error {
	if err := h.deps.VerifyBrowserRequest(c); err != nil {
		return err
	}
	return h.requireAdministrator(c)
}

func (h handler) createConnector(c *echo.Context) error {
	if err := h.requireAdministratorChange(c); err != nil {
		return err
	}
	var request struct {
		DisplayName string `json:"display_name"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "invalid directory connector request")
	}
	grant, err := h.deps.Service.Create(c.Request().Context(), support.RequestTenantID(c), request.DisplayName)
	if errors.Is(err, domain.ErrDisplayNameRequired) {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, grantJSON(grant))
}

func (h handler) listConnectors(c *echo.Context) error {
	if err := h.requireAdministrator(c); err != nil {
		return err
	}
	connectors, err := h.deps.Service.List(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	result := make([]connectorResponse, len(connectors))
	for i, connector := range connectors {
		result[i] = connectorJSON(connector)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"connectors": result})
}

func (h handler) getConnector(c *echo.Context) error {
	if err := h.requireAdministrator(c); err != nil {
		return err
	}
	connector, err := h.deps.Service.Get(c.Request().Context(), support.RequestTenantID(c), c.Param("connector_id"))
	if err != nil {
		return writeAdminError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, connectorJSON(connector))
}

func (h handler) reissueEnrollment(c *echo.Context) error {
	if err := h.requireAdministratorChange(c); err != nil {
		return err
	}
	grant, err := h.deps.Service.ReissueEnrollment(c.Request().Context(), support.RequestTenantID(c), c.Param("connector_id"))
	if err != nil {
		return writeAdminError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, grantJSON(grant))
}

func (h handler) disableConnector(c *echo.Context) error {
	if err := h.requireAdministratorChange(c); err != nil {
		return err
	}
	connector, err := h.deps.Service.Disable(c.Request().Context(), support.RequestTenantID(c), c.Param("connector_id"))
	if err != nil {
		return writeAdminError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, connectorJSON(connector))
}

func writeAdminError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, usecases.ErrNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "directory_connector_not_found", "directory connector not found")
	case errors.Is(err, domain.ErrConnectorDisabled):
		return support.WriteProblem(c, http.StatusConflict, "directory_connector_disabled", err.Error())
	default:
		return err
	}
}

func (h handler) enrollConnector(c *echo.Context) error {
	var request struct {
		EnrollmentToken string         `json:"enrollment_token"`
		JWK             map[string]any `json:"jwk"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "invalid enrollment request")
	}
	connector, err := h.deps.Service.Enroll(c.Request().Context(), support.RequestTenantID(c), request.EnrollmentToken, request.JWK)
	switch {
	case errors.Is(err, usecases.ErrInvalidToken):
		return writeInvalidToken(c)
	case errors.Is(err, usecases.ErrInvalidKey):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "jwk must be a public P-256 or RSA (2048 bits or more) signing key")
	case err != nil:
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"connector_id": connector.ID,
		"key_id":       connector.KeyID,
		"audience":     h.audience(c),
	})
}

func (h handler) reportHeartbeat(c *echo.Context) error {
	assertion, ok := bearer(c)
	if !ok {
		return writeInvalidToken(c)
	}
	err := h.deps.Service.Heartbeat(c.Request().Context(), support.RequestTenantID(c), assertion, h.audience(c))
	if errors.Is(err, usecases.ErrInvalidToken) {
		return writeInvalidToken(c)
	}
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func bearer(c *echo.Context) (string, bool) {
	assertion, ok := strings.CutPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
	return assertion, ok && assertion != ""
}

// authenticateConnector は Connector の JWT を検証し、拒否の応答を書いたかを返す。
func (h handler) authenticateConnector(c *echo.Context) (domain.Connector, bool, error) {
	assertion, ok := bearer(c)
	if !ok {
		return domain.Connector{}, false, writeInvalidToken(c)
	}
	connector, err := h.deps.Service.Authenticate(c.Request().Context(), support.RequestTenantID(c), assertion, h.audience(c))
	if errors.Is(err, usecases.ErrInvalidToken) {
		return domain.Connector{}, false, writeInvalidToken(c)
	}
	return connector, err == nil, err
}

// audience は、要求したテナントの Connector の JWT が aud に持つべき値を返す。
func (h handler) audience(c *echo.Context) string {
	return support.TenantURL(c, usecases.AudiencePath, h.deps.Issuer)
}

func writeInvalidToken(c *echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	return support.WriteProblem(c, http.StatusUnauthorized, "invalid_token", "the directory connector credential is not valid")
}
