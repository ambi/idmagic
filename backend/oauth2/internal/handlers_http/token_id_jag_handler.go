package handlers_http

import (
	"context"
	"net/http"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// applicationAccess は、RP のクライアントの Application への到達を既存の関門で問う。
func (d Deps) applicationAccess(c *echo.Context) tokenusecases.ApplicationAccessCheck {
	if d.ApplicationGate == nil {
		return nil
	}
	clientIP := d.ClientIP(c.Request())
	tenantID := support.RequestTenantID(c)
	return func(ctx context.Context, clientID string, assertion *oauthports.IDTokenAssertion) (bool, error) {
		decision, err := d.EvaluateApplicationAccess(ctx, tenantID, appdomain.ApplicationProtocolOIDC, clientID, assertion.Subject,
			&authdomain.AuthenticationContext{UserID: assertion.Subject, AuthTime: assertion.AuthTime, AMR: assertion.AMR, ACR: assertion.ACR}, clientIP)
		if err != nil {
			return false, err
		}
		return decision.Allowed && !decision.StepUpRequired, nil
	}
}

func (d Deps) issueIDJAG(c *echo.Context, client authedClient, dpopJKT string) error {
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	details, err := tokenusecases.ParseAuthorizationDetails(c.Request().PostFormValue("authorization_details"))
	if err != nil {
		return writeOAuthError(c, err)
	}
	guard, _ := d.TokenIssuer.(oauthports.IssuanceGuard)
	res, err := tokenusecases.IssueIDJAG(ctx, tokenusecases.IssueIDJAGDeps{
		Guard:      guard,
		ClientRepo: d.ClientRepo, UserRepo: d.UserRepo, Policies: d.CrossAppAccessPolicies,
		Verifier: d.IDTokenAssertionVerifier, Signer: d.IDJAGSigner,
		ApplicationAccess: d.applicationAccess(c), UserEpochs: d.UserRevocationEpochRepo, AgentRepo: d.AgentRepo, Consents: d.ConsentRepo, Emit: d.Emit,
	}, tokenusecases.IssueIDJAGInput{
		ClientID:             client.ID,
		SubjectToken:         c.Request().PostFormValue("subject_token"),
		SubjectTokenType:     c.Request().PostFormValue("subject_token_type"),
		HasActorToken:        c.Request().PostFormValue("actor_token") != "",
		Audiences:            c.Request().PostForm["audience"],
		Resources:            c.Request().PostForm["resource"],
		Scope:                c.Request().PostFormValue("scope"),
		AuthorizationDetails: details,
		ProofJKT:             dpopJKT,
		ProofX5TS256:         client.MTLSThumbprintS256,
	}, time.Now().UTC())
	if err != nil {
		return writeOAuthError(c, err)
	}
	body := map[string]any{
		"access_token": res.AccessToken, "issued_token_type": res.IssuedTokenType,
		"token_type": res.TokenType, "expires_in": res.ExpiresIn, "scope": res.Scope,
	}
	if len(res.AuthorizationDetails) > 0 {
		body["authorization_details"] = res.AuthorizationDetails
	}
	return c.JSON(http.StatusOK, body)
}

func (d Deps) redeemIDJAG(c *echo.Context, client authedClient, grant, dpopJKT string) error {
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	res, err := tokenusecases.RedeemIDJAG(ctx, tokenusecases.RedeemIDJAGDeps{
		ClientRepo: d.ClientRepo, UserRepo: d.UserRepo, Policies: d.CrossAppAccessPolicies, Trust: d.IDJAGTrust,
		ResourceServers: d.McpResourceServerRepo, AuthzDetailTypeRepo: d.AuthzDetailTypeRepo,
		Replay: d.IDJAGReplayStore, TokenIssuer: d.TokenIssuer, AgentRepo: d.AgentRepo, Consents: d.ConsentRepo, Emit: d.Emit,
	}, tokenusecases.RedeemIDJAGInput{
		ClientID: client.ID, GrantType: spec.GrantType(grant), Assertion: c.Request().PostFormValue("assertion"),
		SelfIssuer: support.RequestIssuer(c, d.Issuer), ProofJKT: dpopJKT, ProofX5TS256: client.MTLSThumbprintS256,
	}, time.Now().UTC())
	if err != nil {
		return writeOAuthError(c, err)
	}
	body := map[string]any{
		"access_token": res.AccessToken, "token_type": res.TokenType, "expires_in": res.ExpiresIn, "scope": res.Scope,
	}
	if len(res.AuthorizationDetails) > 0 {
		body["authorization_details"] = res.AuthorizationDetails
	}
	return c.JSON(http.StatusOK, body)
}
