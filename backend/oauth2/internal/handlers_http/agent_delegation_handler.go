// /api/account/v1/agent-delegations と /api/admin/v1/agent-delegations — 本人が Agent へ
// 与える代行の付与の操作と、管理者の参照 (REQ-OAUTH2-057、REQ-OAUTH2-058、REQ-OAUTH2-059)。
package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/internal/consent/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type delegableAgentResponse struct {
	AgentID   string `json:"agent_id"`
	Name      string `json:"name"`
	OwnerName string `json:"owner_name,omitempty"`
}

type agentDelegationGrantRequest struct {
	AgentID   string     `json:"agent_id"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type accountAgentDelegationResponse struct {
	AgentID    string              `json:"agent_id"`
	AgentName  string              `json:"agent_name"`
	OwnerName  string              `json:"owner_name,omitempty"`
	Scopes     []string            `json:"scopes"`
	State      domain.ConsentState `json:"state"`
	GrantedAt  time.Time           `json:"granted_at"`
	ExpiresAt  time.Time           `json:"expires_at"`
	LastUsedAt *time.Time          `json:"last_used_at,omitempty"`
}

type adminAgentDelegationResponse struct {
	UserID            string              `json:"user_id"`
	PreferredUsername string              `json:"preferred_username,omitempty"`
	AgentID           string              `json:"agent_id"`
	AgentName         string              `json:"agent_name"`
	Scopes            []string            `json:"scopes"`
	State             domain.ConsentState `json:"state"`
	GrantedAt         time.Time           `json:"granted_at"`
	ExpiresAt         time.Time           `json:"expires_at"`
	RevokedAt         *time.Time          `json:"revoked_at,omitempty"`
	LastUsedAt        *time.Time          `json:"last_used_at,omitempty"`
}

const listAdminAgentDelegationsQuery = "ListAdminAgentDelegations"

func (d Deps) toAccountAgentDelegation(ctx context.Context, tenantID string, grant *domain.Consent, usernames map[string]string) (accountAgentDelegationResponse, error) {
	response := accountAgentDelegationResponse{
		AgentID: grant.AgentID, AgentName: grant.AgentID,
		Scopes: slices.Clone(grant.Scopes), State: grant.State,
		GrantedAt: grant.GrantedAt, ExpiresAt: grant.ExpiresAt, LastUsedAt: grant.LastUsedAt,
	}
	if d.AgentRepo == nil {
		return response, nil
	}
	agent, err := d.AgentRepo.FindByID(ctx, tenantID, grant.AgentID)
	if err != nil {
		return accountAgentDelegationResponse{}, err
	}
	if agent != nil {
		response.AgentName = agent.Name
		response.OwnerName = d.resolveUsername(ctx, usernames, agent.OwnerUserID)
	}
	return response, nil
}

func (d Deps) handleListDelegableAgents(c *echo.Context) error {
	if _, err := d.requireAccountSub(c); err != nil {
		return writeAccountSubError(c, err)
	}
	ctx := c.Request().Context()
	response := []delegableAgentResponse{}
	if d.AgentRepo != nil {
		agents, err := d.AgentRepo.ListAll(ctx, support.RequestTenantID(c))
		if err != nil {
			return err
		}
		usernames := map[string]string{}
		for _, agent := range agents {
			if agent.IsActive() {
				response = append(response, delegableAgentResponse{
					AgentID: agent.ID, Name: agent.Name, OwnerName: d.resolveUsername(ctx, usernames, agent.OwnerUserID),
				})
			}
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"agents": response})
}

func (d Deps) handleListMyAgentDelegations(c *echo.Context) error {
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	ctx := c.Request().Context()
	grants, err := consentusecases.ListGrantedAgentDelegations(ctx, d.ConsentDeps(), sub, time.Now().UTC())
	if err != nil {
		return err
	}
	tenantID := support.RequestTenantID(c)
	usernames := map[string]string{}
	response := make([]accountAgentDelegationResponse, len(grants))
	for i, grant := range grants {
		if response[i], err = d.toAccountAgentDelegation(ctx, tenantID, grant, usernames); err != nil {
			return err
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"delegations": response})
}

func (d Deps) handleGrantMyAgentDelegation(c *echo.Context) error {
	// トークンで付与できると、account:write を持つ Agent が自分自身に付与できる
	// (REQ-OAUTH2-057)。ブラウザーの検証はトークンを素通しするので、先に拒否する。
	if isTokenAuthorization(c.Request().Header.Get("Authorization")) {
		return support.WriteProblem(c, http.StatusForbidden, "access_denied", "Delegation can only be granted from a browser session.")
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	var req agentDelegationGrantRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The request body is invalid.")
	}
	ctx := c.Request().Context()
	grant, err := consentusecases.GrantAgentDelegation(ctx, consentusecases.AgentDelegationDeps{
		ConsentDeps: d.ConsentDeps(), Agents: d.AgentRepo,
	}, consentusecases.AgentDelegationGrant{
		Sub: sub, AgentID: req.AgentID, Scopes: req.Scopes, ExpiresAt: req.ExpiresAt,
	}, time.Now().UTC())
	switch {
	case errors.Is(err, consentusecases.ErrInvalidDelegationGrant):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The scopes or the expiry are invalid.")
	case errors.Is(err, consentusecases.ErrDelegationAgentNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "agent_not_found", "The agent does not exist or is not active.")
	case err != nil:
		return err
	}
	response, err := d.toAccountAgentDelegation(ctx, support.RequestTenantID(c), grant, map[string]string{})
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, response)
}

func isTokenAuthorization(authorization string) bool {
	lowered := strings.ToLower(authorization)
	return strings.HasPrefix(lowered, "bearer ") || strings.HasPrefix(lowered, "dpop ")
}

func (d Deps) handleRevokeMyAgentDelegation(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	// 対象は自分の sub に固定する。URL の agent_id 以外は信用しない。
	if err := consentusecases.RevokeAgentDelegation(c.Request().Context(), d.ConsentDeps(), sub, c.Param("agent_id"), time.Now().UTC()); err != nil {
		return writeConsentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleListAdminAgentDelegations(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	page, err := support.ParsePageRequest(c, d.PaginationCodec, tenantID, listAdminAgentDelegationsQuery, support.DefaultPageLimit, support.MaxPageLimit)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	var grants []*domain.Consent
	if page.Direction == support.PageBackward {
		grants, err = d.ConsentRepo.ListAgentDelegationPageBefore(ctx, tenantID, page.AfterPrimary, page.AfterID, page.Limit+1)
	} else {
		grants, err = d.ConsentRepo.ListAgentDelegationPage(ctx, tenantID, page.AfterPrimary, page.AfterID, page.Limit+1)
	}
	if err != nil {
		return err
	}
	grants, hasPrevious, hasNext := support.TrimPage(grants, page)
	usernames := map[string]string{}
	response := make([]adminAgentDelegationResponse, len(grants))
	for i, grant := range grants {
		account, err := d.toAccountAgentDelegation(ctx, tenantID, grant, usernames)
		if err != nil {
			return err
		}
		response[i] = adminAgentDelegationResponse{
			UserID: grant.UserID, PreferredUsername: d.resolveUsername(ctx, usernames, grant.UserID),
			AgentID: grant.AgentID, AgentName: account.AgentName,
			Scopes: account.Scopes, State: grant.State,
			GrantedAt: grant.GrantedAt, ExpiresAt: grant.ExpiresAt, RevokedAt: grant.RevokedAt, LastUsedAt: grant.LastUsedAt,
		}
	}
	if len(grants) > 0 {
		first := grants[0]
		last := grants[len(grants)-1]
		if err := support.SetPageLinks(c, d.PaginationCodec, d.Issuer, tenantID, listAdminAgentDelegationsQuery,
			first.UserID, first.AgentID, last.UserID, last.AgentID, hasPrevious, hasNext); err != nil {
			return err
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"delegations": response})
}
