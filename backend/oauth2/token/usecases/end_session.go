// /end_session (OIDC RP-Initiated Logout 1.0) の client/session 解決 (ADR-127)。
// local logout (LoginSession revoke / refresh token revoke) と post_logout_redirect_uri
// への redirect は HTTP 層 (end_session_handler.go) が本 use case の結果を使って行う。
package usecases

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

type EndSessionDeps struct {
	ClientRepo   ports.OAuth2ClientRepository
	HintVerifier ports.IDTokenHintVerifier
}

type EndSessionInput struct {
	ClientID              string
	PostLogoutRedirectURI string
	IDTokenHint           string
}

// EndSessionTarget is what ResolveEndSession resolves before any local
// logout or redirect happens.
type EndSessionTarget struct {
	// Sid is the OIDC session id resolved from a verified id_token_hint.
	// Empty when no hint was given; the caller falls back to the browser
	// cookie for session resolution (ADR-127 decision 4).
	Sid string
	// Client / RedirectURI are only populated when PostLogoutRedirectURI was
	// requested. A nil Client means the caller should skip straight to the
	// unauthenticated "signed out" status page (legacy behavior preserved).
	Client      *domain.OAuth2Client
	RedirectURI string
}

// ResolveEndSession は RP-Initiated Logout 1.0 の client 解決・redirect_uri 検証・
// id_token_hint 検証をまとめて行う (ADR-127)。id_token_hint 由来の異常は fail-closed
// で invalid_request として拒否する: 署名検証不能、iss 不一致 (HintVerifier 側で検証)、
// client_id パラメータとの aud 不一致。exp は検証しない。
func ResolveEndSession(ctx context.Context, deps EndSessionDeps, in EndSessionInput) (*EndSessionTarget, error) {
	clientID := in.ClientID
	sid := ""
	if in.IDTokenHint != "" {
		if deps.HintVerifier == nil {
			return nil, NewOAuthError("invalid_request", "id_token_hint はサポートされていません")
		}
		claims, err := deps.HintVerifier.VerifyIDTokenHint(ctx, in.IDTokenHint)
		if err != nil {
			return nil, NewOAuthError("invalid_request", "id_token_hint を検証できません")
		}
		if clientID != "" && clientID != claims.Audience {
			return nil, NewOAuthError("invalid_request", "id_token_hint が client_id と一致しません")
		}
		clientID = claims.Audience
		sid = claims.Sid
	}

	target := &EndSessionTarget{Sid: sid}
	if in.PostLogoutRedirectURI == "" {
		return target, nil
	}
	if clientID == "" {
		return nil, NewOAuthError("invalid_request", "client_id が必要です")
	}
	client, err := deps.ClientRepo.FindByID(ctx, tenancy.TenantID(ctx), clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, NewOAuthError("invalid_request", "post_logout_redirect_uri が未登録")
	}
	registered := ""
	for _, uri := range client.RedirectURIs {
		if uri == in.PostLogoutRedirectURI {
			registered = uri
			break
		}
	}
	if registered == "" {
		return nil, NewOAuthError("invalid_request", "post_logout_redirect_uri が未登録")
	}
	target.Client = client
	target.RedirectURI = registered
	return target, nil
}
