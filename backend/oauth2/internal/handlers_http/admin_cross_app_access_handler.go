// アプリ間アクセスの許可の管理 API (REQ-OAUTH2-072)。テナント境界に閉じる。
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

const (
	crossAppAccessNotFoundCode = "cross_app_access_policy_not_found"
	crossAppAccessExistsCode   = "cross_app_access_policy_exists"
)

type crossAppAccessIssueRequest struct {
	ClientID                 string   `json:"client_id"`
	Audience                 string   `json:"audience"`
	RelyingPartyClientID     string   `json:"relying_party_client_id"`
	TargetClientID           string   `json:"target_client_id"`
	Resources                []string `json:"resources"`
	Scopes                   []string `json:"scopes"`
	AuthorizationDetailTypes []string `json:"authorization_detail_types"`
	ReleaseEmail             bool     `json:"release_email"`
}

type crossAppAccessRedeemRequest struct {
	ProviderID string   `json:"provider_id"`
	ClientID   string   `json:"client_id"`
	Scopes     []string `json:"scopes"`
}

func (d Deps) handleListAdminCrossAppAccessIssuePolicies(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	policies, err := d.CrossAppAccessPolicies.ListIssuePolicies(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"policies": policies})
}

func (d Deps) handleCreateAdminCrossAppAccessIssuePolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req crossAppAccessIssueRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	types := req.AuthorizationDetailTypes
	if types == nil {
		types = []string{}
	}
	policy := &oauthdomain.CrossAppAccessIssuePolicy{
		TenantID: tenantID, ID: id, ClientID: req.ClientID, Audience: req.Audience,
		RelyingPartyClientID: req.RelyingPartyClientID, TargetClientID: req.TargetClientID,
		Resources: req.Resources, Scopes: req.Scopes, AuthorizationDetailTypes: types,
		ReleaseEmail: req.ReleaseEmail, CreatedAt: time.Now().UTC(),
	}
	if err := policy.Validate(); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The policy is invalid.")
	}
	for _, clientID := range []string{policy.ClientID, policy.RelyingPartyClientID} {
		client, err := d.ClientRepo.FindByID(ctx, tenantID, clientID)
		if err != nil {
			return err
		}
		if client == nil {
			return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The client does not exist.")
		}
	}
	if err := d.CrossAppAccessPolicies.CreateIssuePolicy(ctx, policy); err != nil {
		if errors.Is(err, oauthports.ErrCrossAppAccessPolicyExists) {
			return support.WriteProblem(c, http.StatusConflict, crossAppAccessExistsCode, "The same policy is already registered.")
		}
		return err
	}
	d.emitCrossAppAccessCreated(c, policy.ID, oauthdomain.CrossAppAccessDirectionIssue, policy.ClientID)
	return support.NoStoreJSON(c, http.StatusCreated, policy)
}

func (d Deps) handleDeleteAdminCrossAppAccessIssuePolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	policy, err := d.CrossAppAccessPolicies.FindIssuePolicy(ctx, tenantID, c.Param("id"))
	if err != nil {
		return err
	}
	if policy == nil {
		return support.WriteProblem(c, http.StatusNotFound, crossAppAccessNotFoundCode, "The policy does not exist.")
	}
	if err := d.CrossAppAccessPolicies.DeleteIssuePolicy(ctx, tenantID, policy.ID); err != nil {
		return err
	}
	d.emitCrossAppAccessDeleted(c, policy.ID, oauthdomain.CrossAppAccessDirectionIssue, policy.ClientID)
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleListAdminCrossAppAccessRedeemPolicies(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	policies, err := d.CrossAppAccessPolicies.ListRedeemPolicies(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"policies": policies})
}

func (d Deps) handleCreateAdminCrossAppAccessRedeemPolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req crossAppAccessRedeemRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	policy := &oauthdomain.CrossAppAccessRedeemPolicy{
		TenantID: tenantID, ID: id, ProviderID: req.ProviderID, ClientID: req.ClientID, Scopes: req.Scopes, CreatedAt: time.Now().UTC(),
	}
	if err := policy.Validate(); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The policy is invalid.")
	}
	client, err := d.ClientRepo.FindByID(ctx, tenantID, policy.ClientID)
	if err != nil {
		return err
	}
	if client == nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The client does not exist.")
	}
	exists, err := d.IDJAGTrust.ProviderExists(ctx, tenantID, policy.ProviderID)
	if err != nil {
		return err
	}
	if !exists {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The identity provider connection does not exist.")
	}
	if err := d.CrossAppAccessPolicies.CreateRedeemPolicy(ctx, policy); err != nil {
		if errors.Is(err, oauthports.ErrCrossAppAccessPolicyExists) {
			return support.WriteProblem(c, http.StatusConflict, crossAppAccessExistsCode, "The same policy is already registered.")
		}
		return err
	}
	d.emitCrossAppAccessCreated(c, policy.ID, oauthdomain.CrossAppAccessDirectionRedeem, policy.ClientID)
	return support.NoStoreJSON(c, http.StatusCreated, policy)
}

func (d Deps) handleDeleteAdminCrossAppAccessRedeemPolicy(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	policy, err := d.CrossAppAccessPolicies.FindRedeemPolicy(ctx, tenantID, c.Param("id"))
	if err != nil {
		return err
	}
	if policy == nil {
		return support.WriteProblem(c, http.StatusNotFound, crossAppAccessNotFoundCode, "The policy does not exist.")
	}
	if err := d.CrossAppAccessPolicies.DeleteRedeemPolicy(ctx, tenantID, policy.ID); err != nil {
		return err
	}
	d.emitCrossAppAccessDeleted(c, policy.ID, oauthdomain.CrossAppAccessDirectionRedeem, policy.ClientID)
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) emitCrossAppAccessCreated(c *echo.Context, id, direction, clientID string) {
	if d.Emit != nil {
		d.Emit(&oauthdomain.CrossAppAccessPolicyCreated{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), PolicyID: id, Direction: direction, ClientID: clientID,
		})
	}
}

func (d Deps) emitCrossAppAccessDeleted(c *echo.Context, id, direction, clientID string) {
	if d.Emit != nil {
		d.Emit(&oauthdomain.CrossAppAccessPolicyDeleted{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), PolicyID: id, Direction: direction, ClientID: clientID,
		})
	}
}
