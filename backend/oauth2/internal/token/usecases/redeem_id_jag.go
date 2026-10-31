// ID-JAG の償還 (draft-ietf-oauth-identity-assertion-authz-grant-05 §4.4)。クライアント認証の
// client_assertion とは経路を共有しない。一方の緩和がもう一方へ漏れないようにするためである。
package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type RedeemIDJAGDeps struct {
	ClientRepo          ports.OAuth2ClientRepository
	UserRepo            userports.UserRepository
	Policies            ports.CrossAppAccessPolicyRepository
	Trust               ports.IDJAGIssuerTrust
	ResourceServers     ports.McpResourceServerRepository
	AuthzDetailTypeRepo ports.AuthorizationDetailTypeRepository
	Replay              ports.ClientAssertionReplayStore
	TokenIssuer         ports.TokenIssuer
	AgentRepo           agentports.AgentRepository
	Consents            ports.ConsentRepository
	Emit                func(spec.DomainEvent)
}

type RedeemIDJAGInput struct {
	ClientID     string
	GrantType    spec.GrantType
	Assertion    string
	SelfIssuer   string
	ProofJKT     string
	ProofX5TS256 string
}

type RedeemIDJAGResult struct {
	AccessToken          string
	TokenType            string
	ExpiresIn            int
	Scope                string
	AuthorizationDetails []spec.AuthorizationDetail
}

// RedeemIDJAG は REQ-OAUTH2-069〜071 に従って ID-JAG をアクセストークンへ償還する。
func RedeemIDJAG(ctx context.Context, deps RedeemIDJAGDeps, in RedeemIDJAGInput, now time.Time) (*RedeemIDJAGResult, error) {
	tenantID := tenantports.TenantID(ctx)
	reject := func(code, description string) error {
		emit(deps.Emit, &domain.IdJagRejected{At: now, TenantID: tenantID, ClientID: in.ClientID, Stage: idJagStageRedeem, Reason: code})
		return NewOAuthError(code, description)
	}
	invalid := func() error { return reject("invalid_grant", "The assertion is invalid.") }
	denied := func() error { return reject("access_denied", "The access is not permitted.") }

	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil || client.ClientType != spec.ClientConfidential {
		return nil, reject("invalid_client", "A confidential client is required.")
	}
	if in.Assertion == "" {
		return nil, reject("invalid_request", "assertion is required")
	}
	if deps.Trust == nil {
		return nil, invalid()
	}

	issuer, err := tokens_jose.UnverifiedIDJAGIssuer(in.Assertion)
	if err != nil {
		return nil, invalid()
	}
	// §9.4: 自身が発行した ID-JAG では発行しない。
	if issuer == in.SelfIssuer {
		return nil, invalid()
	}
	trusted, err := deps.Trust.ResolveIssuer(ctx, tenantID, issuer)
	if err != nil {
		return nil, err
	}
	if trusted == nil {
		return nil, invalid()
	}
	claims, err := tokens_jose.VerifyIDJAG(in.Assertion, trusted.JWKS, issuer, in.SelfIssuer, now)
	if err != nil {
		return nil, invalid()
	}
	if claims.ClientID != in.ClientID || claims.HasActor {
		return nil, invalid()
	}

	sc, code := decideIDJAGBinding(claims.Confirmation, in.GrantType, in.ProofJKT, in.ProofX5TS256)
	if code != "" {
		return nil, reject(code, "The proof of possession does not satisfy the assertion.")
	}

	if len(claims.Resources) == 0 {
		return nil, invalid()
	}
	var resourceScopes []string
	for _, resource := range claims.Resources {
		server, err := deps.ResourceServers.FindByResource(ctx, tenantID, resource)
		if err != nil {
			return nil, err
		}
		if server == nil || !server.IsActive() {
			return nil, invalid()
		}
		resourceScopes = append(resourceScopes, server.Scopes...)
	}

	policy, err := deps.Policies.FindRedeemPolicyFor(ctx, tenantID, trusted.ProviderID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, denied()
	}
	userID, err := deps.Trust.ResolveSubject(ctx, tenantID, trusted.ProviderID, claims.Subject)
	if err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, denied()
	}
	user, err := deps.UserRepo.FindBySub(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenantID || !user.IsActive() {
		return nil, denied()
	}
	scopes := idJagAccessTokenScopes(strings.Fields(claims.Scope), policy.Scopes, strings.Fields(client.Scope), resourceScopes)
	if len(scopes) == 0 {
		return nil, denied()
	}

	if err := checkIDJAGAgent(ctx, idJagAgentDeps{
		ClientRepo: deps.ClientRepo, AgentRepo: deps.AgentRepo, Consents: deps.Consents, Emit: deps.Emit,
	}, in.ClientID, userID, scopes, now); err != nil {
		if oe, ok := errors.AsType[*OAuthError](err); ok {
			return nil, reject(oe.Code, oe.Description)
		}
		return nil, err
	}

	details, err := registeredAuthorizationDetails(ctx, deps.AuthzDetailTypeRepo, claims.AuthorizationDetails)
	if err != nil {
		return nil, err
	}

	// すべての検証の後で jti を記憶する。proof の失敗で消費すると、クライアントが nonce に
	// 応じて同じ ID-JAG で再試行できない (jwt-dpop-grant-02)。
	window := int(claims.ExpiresAt.Add(tokens_jose.IDJAGClockSkew).Sub(now).Seconds()) + 1
	fresh, err := deps.Replay.RecordIfNew(ctx, idJagReplayKey(issuer, claims.JTI), window, now)
	if err != nil {
		return nil, err
	}
	if !fresh {
		return nil, invalid()
	}

	authTime := claims.AuthTime
	if authTime == 0 {
		authTime = now.Unix()
	}
	token, jti, err := deps.TokenIssuer.SignAccessToken(ctx, ports.AccessTokenInput{
		Client: client, Sub: userID, Scopes: scopes, SenderConstraint: sc, AuthTime: authTime,
		AMR: claims.AMR, ACR: claims.ACR, Audiences: claims.Resources, AuthorizationDetails: details,
	})
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AccessTokenIssued{
		At: now, TenantID: tenantID, JTI: jti, ClientID: client.ClientID, UserID: userID,
		Scopes: scopes, SenderConstraint: senderConstraintTag(sc),
	})
	for _, resource := range claims.Resources {
		emit(deps.Emit, &domain.ResourceScopedTokenIssued{At: now, TenantID: tenantID, ClientID: client.ClientID, Resource: resource, Scopes: scopes})
	}
	emit(deps.Emit, &domain.IdJagRedeemed{
		At: now, TenantID: tenantID, Issuer: issuer, ClientID: client.ClientID, UserID: userID, JTI: claims.JTI,
	})
	return &RedeemIDJAGResult{
		AccessToken: token, TokenType: domain.PresentationTokenType(sc), ExpiresIn: deps.TokenIssuer.AccessTokenTTLSeconds(),
		Scope: strings.Join(scopes, " "), AuthorizationDetails: details,
	}, nil
}

// decideIDJAGBinding は REQ-OAUTH2-071 の表である。第 2 戻り値が空でなければ、その誤りで拒否する。
func decideIDJAGBinding(cnf *tokens_jose.IDJAGConfirmation, grant spec.GrantType, jkt, x5t string) (*domain.SenderConstraint, string) {
	const refused, missingProof = "invalid_grant", "invalid_dpop_proof"
	switch {
	case cnf == nil:
		if grant == spec.GrantJWTDPoP {
			return nil, refused
		}
		return idJagBinding(jkt, x5t), ""
	case cnf.Other || (cnf.JKT != "" && cnf.X5TS256 != ""):
		return nil, refused
	case cnf.JKT != "":
		if grant != spec.GrantJWTDPoP {
			return nil, refused
		}
		if jkt == "" {
			return nil, missingProof
		}
		if jkt != cnf.JKT {
			return nil, refused
		}
		return &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: jkt}, ""
	case cnf.X5TS256 != "":
		if grant != spec.GrantJWTBearer || x5t != cnf.X5TS256 {
			return nil, refused
		}
		return &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: x5t}, ""
	}
	return nil, refused
}

// idJagAccessTokenScopes は ID-JAG のスコープのうち、すべての許可に含まれるものを返す (REQ-OAUTH2-070)。
func idJagAccessTokenScopes(requested, policy, client, resources []string) []string {
	var out []string
	for _, scope := range requested {
		if slices.Contains(policy, scope) && slices.Contains(client, scope) && slices.Contains(resources, scope) && !slices.Contains(out, scope) {
			out = append(out, scope)
		}
	}
	return out
}

// registeredAuthorizationDetails は登録した型で検証を通る詳細だけを残す。読めない詳細は
// 権限を広げないよう捨てる。
func registeredAuthorizationDetails(ctx context.Context, repo ports.AuthorizationDetailTypeRepository, raw json.RawMessage) ([]spec.AuthorizationDetail, error) {
	if len(raw) == 0 || repo == nil {
		return nil, nil
	}
	var details []spec.AuthorizationDetail
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, nil //nolint:nilerr // 発行者が決めた形が読めなければ載せない。
	}
	var out []spec.AuthorizationDetail
	for _, detail := range details {
		err := ValidateAuthorizationDetails(ctx, repo, []spec.AuthorizationDetail{detail})
		if err == nil {
			out = append(out, detail)
			continue
		}
		if _, ok := errors.AsType[*OAuthError](err); !ok {
			return nil, err
		}
	}
	return out, nil
}

// idJagReplayKey は発行者ごとに jti を分け、長さを揃える。jti は外部の発行者が決める値である。
func idJagReplayKey(issuer, jti string) string {
	digest := sha256.Sum256([]byte(issuer + "\x00" + jti))
	return hex.EncodeToString(digest[:])
}
