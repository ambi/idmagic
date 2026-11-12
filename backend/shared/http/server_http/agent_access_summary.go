package server_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type agentAccessSummary struct {
	AsOf              time.Time               `json:"as_of"`
	LastTokenIssuedAt *time.Time              `json:"last_token_issued_at,omitempty"`
	Clients           []agentAccessClient     `json:"clients"`
	Delegations       []agentAccessDelegation `json:"delegations"`
}

type agentAccessClient struct {
	ClientID string   `json:"client_id"`
	Name     *string  `json:"name,omitempty"`
	Scopes   []string `json:"scopes"`
	Missing  bool     `json:"missing"`
}

type agentAccessDelegation struct {
	UserID    string                   `json:"user_id"`
	Scopes    []string                 `json:"scopes"`
	State     oauthdomain.ConsentState `json:"state"`
	GrantedAt time.Time                `json:"granted_at"`
	ExpiresAt time.Time                `json:"expires_at"`
	RevokedAt *time.Time               `json:"revoked_at,omitempty"`
	Active    bool                     `json:"active"`
}

// 登録された材料の読み取りだけを合成する。操作割当と条件の評価はそれぞれの所有者に残す。
func registerAgentAccessSummary(g *echo.Group, d Deps, authenticator *support.Authenticator) {
	g.GET("/api/admin/v1/agents/:agent_id/access-summary", func(c *echo.Context) error {
		return handleGetAgentAccessSummary(d, authenticator, c)
	})
}

func handleGetAgentAccessSummary(d Deps, authenticator *support.Authenticator, c *echo.Context) error {
	if _, err := authenticator.RequireAdmin(c); err != nil {
		return authenticator.WriteAdminAccessError(c, err)
	}
	ctx, tenantID, agentID := c.Request().Context(), support.RequestTenantID(c), c.Param("agent_id")
	if d.IdManagement.AgentRepo == nil {
		return errors.New("agent access summary requires an agent repository")
	}
	agent, err := d.IdManagement.AgentRepo.FindByID(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return support.WriteProblem(c, http.StatusNotFound, "agent_not_found", "The agent does not exist.")
	}
	if d.OAuth2.ClientRepo == nil || d.OAuth2.ConsentRepo == nil || d.Audit.AuditEventRepo == nil {
		return errors.New("agent access summary requires client, consent, and audit repositories")
	}
	result := agentAccessSummary{AsOf: time.Now().UTC(), Clients: []agentAccessClient{}, Delegations: []agentAccessDelegation{}}
	bindings, err := d.IdManagement.AgentRepo.ListBindings(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		client, err := d.OAuth2.ClientRepo.FindByID(ctx, tenantID, binding.ClientID)
		if err != nil {
			return err
		}
		entry := agentAccessClient{ClientID: binding.ClientID, Scopes: []string{}, Missing: client == nil}
		if client != nil {
			entry.Name, entry.Scopes = client.ClientName, strings.Fields(client.Scope)
		}
		result.Clients = append(result.Clients, entry)
	}
	grants, err := d.OAuth2.ConsentRepo.ListDelegationsForAgent(ctx, tenantID, agentID)
	if err != nil {
		return err
	}
	for _, grant := range grants {
		result.Delegations = append(result.Delegations, agentAccessDelegation{
			UserID: grant.UserID, Scopes: grant.Scopes, State: grant.State,
			GrantedAt: grant.GrantedAt, ExpiresAt: grant.ExpiresAt, RevokedAt: grant.RevokedAt,
			Active: grant.PermitsDelegation(nil, result.AsOf),
		})
	}
	issued, err := d.Audit.AuditEventRepo.LatestByAgent(ctx, tenantID, []string{agentID}, tokenIssuanceEventTypes)
	if err != nil {
		return err
	}
	if last, ok := issued[agentID]; ok {
		result.LastTokenIssuedAt = &last
	}
	return support.NoStoreJSON(c, http.StatusOK, result)
}
