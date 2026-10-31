// /token (5 grant types) + /revoke + /introspect
package handlers_http

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	approvalusecases "github.com/ambi/idmagic/backend/oauth2/internal/approval/usecases"
	deviceusecases "github.com/ambi/idmagic/backend/oauth2/internal/device/usecases"
	policytenancy "github.com/ambi/idmagic/backend/oauth2/internal/policy_tenancy"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// handleToken dispatches by grant_type and records the token issuance golden
// signal (oauth2_token_issuance_total/_duration_seconds{grant_type,outcome})
// exactly once per call, regardless of which branch or error path the grant
// dispatch takes, by observing the final response status after dispatchToken
// returns rather than instrumenting every return statement individually.
func (d Deps) handleToken(c *echo.Context) error {
	grantType := c.Request().PostFormValue("grant_type")
	start := time.Now()
	err := d.dispatchToken(c)
	if d.Metrics != nil {
		outcome := "error"
		if response, ok := c.Response().(*echo.Response); ok && response.Status >= 200 && response.Status < 300 {
			outcome = "success"
		}
		if grantType == "" {
			grantType = "unknown"
		}
		d.Metrics.RecordTokenIssuance(grantType, outcome, time.Since(start))
	}
	return err
}

func (d Deps) dispatchToken(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	clientStub, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	clientIP := extractClientIP(c.Request(), d.TrustedForwardedHops)
	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "token", clientStub.ID+"|"+clientIP); err != nil {
		return err
	} else if blocked {
		return nil
	}
	grantType := c.Request().PostFormValue("grant_type")
	if grantType == "" {
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_request", "grant_type is required"))
	}
	if !spec.GrantType(grantType).Valid() {
		return writeOAuthError(c, tokenusecases.NewOAuthError("unsupported_grant_type", "Unsupported grant_type: "+grantType))
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), clientStub.ID)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if client == nil {
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_client", "unknown client_id"))
	}
	// Agent のクライアントの認証方式は、送信者制約と同じ理由でグラントの分岐より前に
	// 1 度だけ判定する。Agent を解決しないグラント (認可コードなど) も Agent の資格情報で
	// 認証していることは変わらない (AIMS00-AGENT-CLIENT-AUTH)。
	if err := tokenusecases.RejectAgentClientWithoutKeyProof(c.Request().Context(), tokenusecases.AgentIssuanceDeps{
		AgentRepo: d.AgentRepo, Emit: d.Emit,
	}, client, grantType, time.Now().UTC()); err != nil {
		return writeOAuthError(c, err)
	}
	// 要求したクライアントの Agent も同じ理由でグラントの分岐より前に 1 度だけ引き、どの
	// グラントで発行したトークンにも発行時の Agent として載せる (REQ-OAUTH2-012)。以降の
	// d.TokenIssuer はこの要求に限った発行器である。
	requestingAgentID, err := tokenusecases.RequestingAgentID(c.Request().Context(), tokenusecases.AgentIssuanceDeps{
		AgentRepo: d.AgentRepo,
	}, client.ClientID)
	if err != nil {
		return writeOAuthError(c, err)
	}
	d.TokenIssuer = tokenusecases.IssuerForRequestingAgent(d.TokenIssuer, requestingAgentID)
	if !slices.Contains(client.GrantTypes, spec.GrantType(grantType)) {
		return writeOAuthError(c, tokenusecases.NewOAuthError("unauthorized_client", "undeclared grant_type"))
	}
	if grantType == "client_credentials" && slices.ContainsFunc(strings.Fields(c.Request().PostFormValue("scope")), spec.IsAccountScope) {
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_scope", "account scope requires user subject"))
	}

	// DPoP 検証 (任意)
	var dpopJKT string
	if proof := c.Request().Header.Get("DPoP"); proof != "" && d.DpopReplayStore != nil {
		htu := support.RequestHTU(c, d.Issuer)
		// No ath is required here: the access token it would bind to does not exist
		// at token request time (RFC 9449 §4.3).
		r, err := tokens_jose.VerifyDPoPForToken(c.Request().Context(), proof, "POST", htu, d.DpopReplayStore, time.Now().UTC())
		if err != nil {
			return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_dpop_proof", err.Error()))
		}
		if r != nil {
			dpopJKT = r.JKT
		}
	}

	// 送信者制約の判定はグラント種別の分岐より前に 1 度だけ置く。分岐の後に置くと、
	// グラントを足すたびに同じ判断を書き写すことになり、書き忘れた経路だけ制約が
	// 消える (FAPI2-SENDER-CONSTRAINT)。
	if !client.SenderConstraintSatisfied(dpopJKT, clientStub.MTLSThumbprintS256) {
		return writeOAuthError(c, tokenusecases.NewOAuthError(
			"invalid_request", "this client requires a DPoP proof or a mutual-TLS certificate"))
	}

	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	now := time.Now().UTC()

	switch grantType {
	case "authorization_code":
		out, err := tokenusecases.ExchangeCodeForToken(ctx, tokenusecases.ExchangeCodeDeps{
			ClientRepo: d.ClientRepo, UserRepo: d.UserRepo,
			RequestStore: d.RequestStore, CodeStore: d.CodeStore,
			RefreshStore: d.RefreshStore, TokenIssuer: d.TokenIssuer,
			ClientSessionStore: d.ClientSessionStore,
			Emit:               d.Emit, ResolveAttributeDefs: d.effectiveUserAttributeDefs,
		}, tokenusecases.ExchangeCodeInput{
			ClientID: clientStub.ID, Code: c.Request().PostFormValue("code"),
			CodeVerifier: c.Request().PostFormValue("code_verifier"),
			RedirectURI:  c.Request().PostFormValue("redirect_uri"),
			DpopJKT:      dpopJKT,
			MTLSX5TS256:  clientStub.MTLSThumbprintS256,
			Resource:     c.Request().PostForm["resource"],
		})
		if err != nil {
			return writeOAuthError(c, err)
		}
		body := map[string]any{
			"access_token": out.AccessToken,
			"token_type":   out.TokenType, "expires_in": out.ExpiresIn, "scope": out.Scope,
		}
		if out.IDToken != "" {
			body["id_token"] = out.IDToken
		}
		if out.RefreshToken != "" {
			body["refresh_token"] = out.RefreshToken
		}
		return c.JSON(http.StatusOK, body)

	case "refresh_token":
		rt := c.Request().PostFormValue("refresh_token")
		if rt == "" {
			return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_request", "refresh_token is required"))
		}
		res, err := tokenusecases.RefreshTokens(ctx, tokenusecases.RefreshDeps{
			ClientRepo: d.ClientRepo, UserRepo: d.UserRepo,
			RefreshStore: d.RefreshStore, UserEpochs: d.UserRevocationEpochRepo, TokenIssuer: d.TokenIssuer,
			Authorizer: d.Authorizer, Emit: d.Emit,
		}, tokenusecases.RefreshInput{
			ClientID: clientStub.ID, RefreshToken: rt,
			ProofJKT: dpopJKT, ProofX5TS256: clientStub.MTLSThumbprintS256,
		}, now)
		if err != nil {
			return writeOAuthError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{
			"access_token": res.AccessToken, "refresh_token": res.RefreshToken,
			"token_type": res.TokenType, "expires_in": res.ExpiresIn, "scope": res.Scope,
		})

	case "client_credentials":
		if client.ClientType != spec.ClientConfidential {
			return writeOAuthError(c, tokenusecases.NewOAuthError("unauthorized_client", "public client not allowed"))
		}
		scope := c.Request().PostFormValue("scope")
		if scope == "" {
			scope = client.Scope
		}
		scopes := strings.Fields(scope)
		declared := map[string]bool{}
		for s := range strings.FieldsSeq(client.Scope) {
			declared[s] = true
		}
		for _, s := range scopes {
			if !declared[s] {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_scope", "The requested scope is not declared."))
			}
		}
		// RFC 8707 resource indicator (wi-262) — 指定時は登録済み Active な
		// McpResourceServer に audience を限定する。未指定時は従来どおり client_id。
		mcpResourceServer, err := tokenusecases.ResolveResourceIndicator(ctx, d.McpResourceServerRepo, support.RequestTenantID(c), c.Request().PostForm["resource"], scopes)
		if err != nil {
			return writeOAuthError(c, err)
		}
		var audiences []string
		if mcpResourceServer != nil {
			audiences = []string{mcpResourceServer.Resource}
		}
		var sc *domain.SenderConstraint
		if dpopJKT != "" {
			sc = &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: dpopJKT}
		} else if clientStub.MTLSThumbprintS256 != "" {
			sc = &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: clientStub.MTLSThumbprintS256}
		}
		// client に Agent が束縛されている場合、Agent 自身が Active でない
		// (Disabled / Killed) か、所有者がオフボードされていれば新規トークンを
		// 発行しない (fail-closed)。束縛があれば agent_id を token に載せる。
		// client_credentials は人間の承認を記録しないため、Autonomous と確認できた
		// Agent だけが通る (REQ-OAUTH2-050)。
		agent, err := tokenusecases.ResolveIssuableAgentWithoutApproval(ctx, tokenusecases.AgentIssuanceDeps{
			AgentRepo: d.AgentRepo, UserRepo: d.UserRepo, Emit: d.Emit,
		}, client.ClientID, grantType, now)
		if err != nil {
			return writeOAuthError(c, err)
		}
		var agentID string
		if agent != nil {
			agentID = agent.ID
		}
		token, jti, err := d.TokenIssuer.SignAccessToken(ctx, oauthports.AccessTokenInput{
			Client: client, Sub: client.ClientID, Scopes: scopes,
			SenderConstraint: sc, AuthTime: now.Unix(), AgentID: agentID,
			Audiences: audiences,
		})
		if err != nil {
			return writeOAuthError(c, err)
		}
		if d.Emit != nil {
			tag := "none"
			if sc != nil {
				tag = string(sc.Type)
			}
			d.Emit(&domain.AccessTokenIssued{At: now, TenantID: support.RequestTenantID(c), JTI: jti, ClientID: client.ClientID, UserID: client.ClientID, Scopes: scopes, SenderConstraint: tag, AgentID: agentID})
			if mcpResourceServer != nil {
				d.Emit(&domain.ResourceScopedTokenIssued{At: now, TenantID: support.RequestTenantID(c), ClientID: client.ClientID, Resource: mcpResourceServer.Resource, Scopes: scopes})
			}
		}
		tokenType := domain.PresentationTokenType(sc)
		return c.JSON(http.StatusOK, map[string]any{
			"access_token": token, "token_type": tokenType,
			"expires_in": d.TokenIssuer.AccessTokenTTLSeconds(), "scope": strings.Join(scopes, " "),
		})

	case "urn:ietf:params:oauth:grant-type:device_code":
		dc := c.Request().PostFormValue("device_code")
		if dc == "" {
			return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_request", "device_code is required"))
		}
		res, err := deviceusecases.ExchangeDeviceCode(ctx, deviceusecases.ExchangeDeviceCodeDeps{
			ClientRepo: d.ClientRepo, UserRepo: d.UserRepo,
			DeviceCodeStore: d.DeviceCodeStore, RefreshStore: d.RefreshStore,
			TokenIssuer:           d.TokenIssuer,
			McpResourceServerRepo: d.McpResourceServerRepo,
			Emit:                  d.Emit,
			ResolveAttributeDefs:  d.effectiveUserAttributeDefs,
		}, deviceusecases.ExchangeDeviceCodeInput{
			ClientID: clientStub.ID, DeviceCode: dc,
			ProofJKT: dpopJKT, ProofX5TS256: clientStub.MTLSThumbprintS256,
			Resource: c.Request().PostForm["resource"],
		}, now)
		if err != nil {
			return writeOAuthError(c, err)
		}
		body := map[string]any{
			"access_token": res.AccessToken, "token_type": res.TokenType,
			"expires_in": res.ExpiresIn, "scope": res.Scope,
		}
		if res.RefreshToken != "" {
			body["refresh_token"] = res.RefreshToken
		}
		if res.IDToken != "" {
			body["id_token"] = res.IDToken
		}
		return c.JSON(http.StatusOK, body)

	case "urn:openid:params:grant-type:ciba":
		result, err := approvalusecases.ExchangeApproval(ctx, approvalusecases.ExchangeApprovalDeps{
			ClientRepo: d.ClientRepo, UserRepo: d.UserRepo, AgentRepo: d.AgentRepo,
			Store: d.ApprovalRequestStore, TokenIssuer: d.TokenIssuer,
			ResolveAttributeDefs: d.effectiveUserAttributeDefs, ResourceServers: d.McpResourceServerRepo, Emit: d.Emit,
		}, approvalusecases.ExchangeApprovalInput{
			ClientID: clientStub.ID, AuthReqID: c.Request().PostFormValue("auth_req_id"),
			ProofJKT: dpopJKT, ProofX5TS256: clientStub.MTLSThumbprintS256,
		}, now)
		if err != nil {
			return writeOAuthError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{
			"access_token": result.AccessToken, "id_token": result.IDToken,
			"token_type": result.TokenType, "expires_in": result.ExpiresIn, "scope": result.Scope,
		})

	case string(spec.GrantJWTBearer), string(spec.GrantJWTDPoP):
		if d.CrossAppAccessPolicies == nil || d.IDJAGReplayStore == nil {
			return writeOAuthError(c, tokenusecases.NewOAuthError("unsupported_grant_type", "Unsupported grant_type: "+grantType))
		}
		return d.redeemIDJAG(c, clientStub, grantType, dpopJKT)

	case "urn:ietf:params:oauth:grant-type:token-exchange":
		if c.Request().PostFormValue("requested_token_type") == tokenusecases.TokenTypeIDJAGURN {
			if d.CrossAppAccessPolicies == nil || d.IDJAGSigner == nil || d.IDTokenAssertionVerifier == nil {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_request", "unsupported requested_token_type"))
			}
			return d.issueIDJAG(c, clientStub, dpopJKT)
		}
		exchangeDetails, err := tokenusecases.ParseAuthorizationDetails(c.Request().PostFormValue("authorization_details"))
		if err != nil {
			return writeOAuthError(c, err)
		}
		res, err := tokenusecases.ExchangeToken(ctx, tokenusecases.ExchangeTokenDeps{
			// 入力のトークンは /introspect と同じ失効判定を通す (RFC8693-SUBJECT-TOKEN)。
			ClientRepo: d.ClientRepo, Introspector: tokenusecases.RevocationCheckingIntrospector{Deps: d.introspectDeps()},
			TokenIssuer: d.TokenIssuer, Authorizer: d.Authorizer,
			AuthzDetailTypeRepo: d.AuthzDetailTypeRepo, McpResourceServerRepo: d.McpResourceServerRepo,
			AgentRepo: d.AgentRepo, Consents: d.ConsentRepo,
			WorkloadVerifier: d.WorkloadVerifier, Emit: d.Emit,
			DelegationPolicy: policytenancy.DelegationPolicyResolver{Tenants: d.TenantRepo},
		}, tokenusecases.ExchangeTokenInput{
			ClientID:             clientStub.ID,
			SubjectToken:         c.Request().PostFormValue("subject_token"),
			SubjectTokenType:     c.Request().PostFormValue("subject_token_type"),
			ActorToken:           c.Request().PostFormValue("actor_token"),
			ActorTokenType:       c.Request().PostFormValue("actor_token_type"),
			Resource:             c.Request().PostForm["resource"],
			Scope:                c.Request().PostFormValue("scope"),
			RequestedTokenType:   c.Request().PostFormValue("requested_token_type"),
			ProofJKT:             dpopJKT,
			ProofX5TS256:         clientStub.MTLSThumbprintS256,
			AuthorizationDetails: exchangeDetails,
		}, now)
		if err != nil {
			return writeOAuthError(c, err)
		}
		body := map[string]any{
			"access_token":      res.AccessToken,
			"issued_token_type": res.IssuedTokenType,
			"token_type":        res.TokenType,
			"expires_in":        res.ExpiresIn,
			"scope":             res.Scope,
		}
		if len(res.AuthorizationDetails) > 0 {
			body["authorization_details"] = res.AuthorizationDetails
		}
		return c.JSON(http.StatusOK, body)
	}
	return writeOAuthError(c, tokenusecases.NewOAuthError("unsupported_grant_type", "Unsupported grant_type: "+grantType))
}

func (d Deps) handleRevoke(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	client := authedClient{}
	var err error
	if builtinID := d.BuiltinResource.RevocationClientID; builtinID != "" &&
		c.Request().PostFormValue("client_id") == builtinID &&
		c.Request().Header.Get("Authorization") == "" && c.Request().PostFormValue("client_secret") == "" {
		client.ID = builtinID
	} else {
		client, err = d.authenticateTokenClient(c)
		if err != nil {
			return writeOAuthError(c, err)
		}
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	if err := tokenusecases.RevokeToken(ctx, tokenusecases.RevokeDeps{
		RefreshStore: d.RefreshStore, Introspector: d.TokenIntrospector,
		AccessTokenDenylist: d.AccessTokenDenylist, Emit: d.Emit,
		ManagedTokenRevoker: d.ManagedTokenRevoker,
	}, client.ID, c.Request().PostFormValue("token"), time.Now().UTC()); err != nil {
		return writeOAuthError(c, err)
	}
	return c.NoContent(http.StatusOK)
}

// introspectDeps は /introspect とトークン交換の入力の検証が共有する失効判定の依存である。
func (d Deps) introspectDeps() tokenusecases.IntrospectDeps {
	return tokenusecases.IntrospectDeps{
		Introspector: d.TokenIntrospector, RefreshStore: d.RefreshStore,
		AccessTokenDenylist: d.AccessTokenDenylist,
		AgentRepo:           d.AgentRepo, RevocationEpochRepo: d.RevocationEpochRepo,
		Consents: d.ConsentRepo, Clients: d.ClientRepo,
		Users: d.UserRepo, UserEpochs: d.UserRevocationEpochRepo,
	}
}

func (d Deps) handleIntrospect(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	clientStub, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	resp, err := tokenusecases.IntrospectToken(c.Request().Context(), d.introspectDeps(), tokenusecases.IntrospectInput{
		Token:         c.Request().PostFormValue("token"),
		TokenTypeHint: c.Request().PostFormValue("token_type_hint"),
	}, time.Now().UTC())
	if err != nil {
		return writeOAuthError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.TokenIntrospected{At: time.Now().UTC(), TenantID: support.RequestTenantID(c), RSClientID: clientStub.ID, TokenID: resp.JTI, Active: resp.Active})
	}
	return c.JSON(http.StatusOK, resp)
}
