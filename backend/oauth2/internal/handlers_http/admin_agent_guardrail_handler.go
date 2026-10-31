// Agent のガードレールの管理 API (REQ-OAUTH2-077)。参照は agents:read、変更は agents:write で守る。
package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type agentGuardrailRequest struct {
	oauthdomain.AgentGuardrailLimits
	ExpectedVersion *string `json:"expected_version"`
}

type agentGuardrailResponse struct {
	oauthdomain.AgentGuardrailLimits
	AgentID   string    `json:"agent_id,omitempty"`
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type agentGuardrailSetResponse struct {
	Agent         *agentGuardrailResponse `json:"agent,omitempty"`
	TenantDefault *agentGuardrailResponse `json:"tenant_default,omitempty"`
}

func toAgentGuardrailResponse(policy *oauthdomain.AgentGuardrailPolicy) *agentGuardrailResponse {
	if policy == nil {
		return nil
	}
	return &agentGuardrailResponse{AgentGuardrailLimits: policy.AgentGuardrailLimits, AgentID: policy.AgentID, Version: policy.Version(), UpdatedAt: policy.UpdatedAt}
}

// agentInTenant は対象の Agent が要求のテナントにあるかを返す。別のテナントの Agent は存在しないものとして扱う。
func (d Deps) agentInTenant(ctx context.Context, tenantID, agentID string) (bool, error) {
	if d.AgentRepo == nil {
		return false, nil
	}
	agent, err := d.AgentRepo.FindByID(ctx, tenantID, agentID)
	return agent != nil, err
}

func writeAgentNotFound(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "agent_not_found", "The agent does not exist.")
}

func (d Deps) handleGetAdminAgentGuardrailPolicy(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx, tenantID, agentID := c.Request().Context(), support.RequestTenantID(c), c.Param("agent_id")
	found, err := d.agentInTenant(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	if !found {
		return writeAgentNotFound(c)
	}
	policy, err := d.AgentGuardrailPolicies.Find(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	tenantDefault, err := d.AgentGuardrailPolicies.Find(ctx, tenantID, "")
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, agentGuardrailSetResponse{Agent: toAgentGuardrailResponse(policy), TenantDefault: toAgentGuardrailResponse(tenantDefault)})
}

func (d Deps) handleGetAdminAgentGuardrailDefault(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	tenantDefault, err := d.AgentGuardrailPolicies.Find(c.Request().Context(), support.RequestTenantID(c), "")
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, agentGuardrailSetResponse{TenantDefault: toAgentGuardrailResponse(tenantDefault)})
}

func (d Deps) handlePutAdminAgentGuardrailPolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	principal, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	agentID := c.Param("agent_id")
	found, err := d.agentInTenant(c.Request().Context(), support.RequestTenantID(c), agentID)
	if err != nil {
		return err
	}
	if !found {
		return writeAgentNotFound(c)
	}
	return d.saveAgentGuardrail(c, principal.ID, agentID)
}

func (d Deps) handlePutAdminAgentGuardrailDefault(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	principal, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.saveAgentGuardrail(c, principal.ID, "")
}

func (d Deps) saveAgentGuardrail(c *echo.Context, actorUserID, agentID string) error {
	var req agentGuardrailRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil || req.ExpectedVersion == nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body with expected_version is required.")
	}
	if err := req.Validate(); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_agent_guardrail", "The guardrail policy is invalid.")
	}
	tenantID := support.RequestTenantID(c)
	now := time.Now().UTC()
	policy := &oauthdomain.AgentGuardrailPolicy{TenantID: tenantID, AgentID: agentID, AgentGuardrailLimits: req.AgentGuardrailLimits, UpdatedAt: now}
	err := d.AgentGuardrailPolicies.Save(c.Request().Context(), policy, *req.ExpectedVersion)
	if errors.Is(err, oauthports.ErrAgentGuardrailConflict) {
		return support.WriteProblem(c, http.StatusConflict, "agent_guardrail_conflict", "The guardrail policy was changed by another request.")
	}
	if err != nil {
		return err
	}
	if d.Emit != nil {
		d.Emit(&oauthdomain.AgentGuardrailPolicyChanged{At: now, TenantID: tenantID, AgentID: agentID, ActorUserID: actorUserID, Version: policy.Version(), Limits: &policy.AgentGuardrailLimits})
	}
	return support.NoStoreJSON(c, http.StatusOK, toAgentGuardrailResponse(policy))
}

func (d Deps) handleDeleteAdminAgentGuardrailPolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	principal, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	agentID := c.Param("agent_id")
	found, err := d.agentInTenant(c.Request().Context(), support.RequestTenantID(c), agentID)
	if err != nil {
		return err
	}
	if !found {
		return writeAgentNotFound(c)
	}
	return d.deleteAgentGuardrail(c, principal.ID, agentID)
}

func (d Deps) handleDeleteAdminAgentGuardrailDefault(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	principal, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.deleteAgentGuardrail(c, principal.ID, "")
}

func (d Deps) deleteAgentGuardrail(c *echo.Context, actorUserID, agentID string) error {
	tenantID := support.RequestTenantID(c)
	if err := d.AgentGuardrailPolicies.Delete(c.Request().Context(), tenantID, agentID); err != nil {
		return err
	}
	if d.Emit != nil {
		d.Emit(&oauthdomain.AgentGuardrailPolicyChanged{At: time.Now().UTC(), TenantID: tenantID, AgentID: agentID, ActorUserID: actorUserID})
	}
	return c.NoContent(http.StatusNoContent)
}
