// /resource-access/evaluate (REQ-OAUTH2-062)
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type resourceAccessResponse struct {
	Decision             bool                       `json:"decision"`
	Reasons              []string                   `json:"reasons,omitempty"`
	Scope                string                     `json:"scope,omitempty"`
	DelegationMode       domain.DelegationMode      `json:"delegation_mode,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
}

func (d Deps) handleEvaluateResourceAccess(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return support.NoStoreJSON(c, http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	client, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	form := c.Request().PostFormValue
	in := tokenusecases.ResourceAccessRequest{
		TenantID: support.RequestTenantID(c), ClientID: client.ID, Token: form("token"),
		ResourceType: form("resource_type"), ResourceID: form("resource_id"), Relation: form("relation"),
		DPoPProof: form("dpop_proof"), HTM: form("htm"), HTU: form("htu"),
		CertificateThumbprint: form("client_certificate_thumbprint"),
	}
	if in.Token == "" || in.ResourceType == "" || in.ResourceID == "" || in.Relation == "" {
		return support.NoStoreJSON(c, http.StatusBadRequest,
			support.OAuthErrorBody("invalid_request", "token, resource_type, resource_id and relation are required"))
	}
	decision, err := tokenusecases.EvaluateResourceAccess(c.Request().Context(), tokenusecases.ResourceAccessDeps{
		Servers: d.McpResourceServerRepo,
		Tokens:  tokenusecases.RevocationCheckingIntrospector{Deps: d.introspectDeps()},
		Clients: d.ClientRepo, Relations: d.RelationChecker, Replay: d.DpopReplayStore, Emit: d.Emit,
	}, in, time.Now().UTC())
	if errors.Is(err, tokenusecases.ErrResourceServerNotRegistered) {
		return support.NoStoreJSON(c, http.StatusForbidden,
			support.OAuthErrorBody("access_denied", "the client does not represent an active resource server"))
	}
	if err != nil {
		// 評価できないことを許可へ倒さない。原因はリソースサーバーへ語らない。
		c.Logger().Error("resource access evaluation failed", "error", err)
		return support.NoStoreJSON(c, http.StatusServiceUnavailable,
			support.OAuthErrorBody("temporarily_unavailable", "the evaluation could not complete"))
	}
	return support.NoStoreJSON(c, http.StatusOK, resourceAccessResponse{
		Decision: decision.Permitted, Reasons: decision.Reasons, Scope: decision.Scope,
		DelegationMode: decision.DelegationMode, AuthorizationDetails: decision.AuthorizationDetails,
	})
}
