// /end_session (OIDC RP-Initiated Logout 1.0) の client/session 解決。
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
	// SessionOwner は id_token_hint の sid が指す LoginSession の主体を返す。
	// ローカル失効を行わない呼び出し元では nil でよい。
	SessionOwner ports.LoginSessionOwnerLookup
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
	// cookie for session resolution (decision 4).
	Sid     string
	Subject string
	// Client / RedirectURI are only populated when PostLogoutRedirectURI was
	// requested. A nil Client means the caller should skip straight to the
	// unauthenticated "signed out" status page (legacy behavior preserved).
	Client      *domain.OAuth2Client
	RedirectURI string
}

// ResolveEndSession は RP-Initiated Logout 1.0 の client 解決・redirect_uri 検証・
// id_token_hint 検証をまとめて行う。id_token_hint 由来の異常は fail-closed
// で invalid_request として拒否する: 署名検証不能、iss 不一致、sub / aud の欠落
// (いずれも HintVerifier 側で検証)、client_id パラメータとの aud 不一致、sid の欠落、
// sid が指す LoginSession の主体と sub の不一致。exp は検証しない。
func ResolveEndSession(ctx context.Context, deps EndSessionDeps, in EndSessionInput) (*EndSessionTarget, error) {
	clientID := in.ClientID
	sid := ""
	subject := ""
	if in.IDTokenHint != "" {
		if deps.HintVerifier == nil {
			return nil, NewOAuthError("invalid_request", "id_token_hint is not supported.")
		}
		claims, err := deps.HintVerifier.VerifyIDTokenHint(ctx, in.IDTokenHint)
		if err != nil {
			return nil, NewOAuthError("invalid_request", "failed to verify id_token_hint")
		}
		// sid の無い hint はログアウト対象を名指ししていない。ここで通すと呼び出し元が
		// browser cookie のセッションへ降格し、hint が指していない対象を失効させる。
		// CIBA は sid を持たない ID Token も hint に使うため、この条件はログアウト側が持つ。
		if claims.Sid == "" {
			return nil, NewOAuthError("invalid_request", "id_token_hint has no sid")
		}
		if clientID != "" && clientID != claims.Audience {
			return nil, NewOAuthError("invalid_request", "id_token_hint does not match client_id")
		}
		if err := checkSessionSubject(ctx, deps.SessionOwner, claims.Sid, claims.Subject); err != nil {
			return nil, err
		}
		clientID = claims.Audience
		sid = claims.Sid
		subject = claims.Subject
	}

	target := &EndSessionTarget{Sid: sid, Subject: subject}
	if in.PostLogoutRedirectURI == "" {
		return target, nil
	}
	if clientID == "" {
		return nil, NewOAuthError("invalid_request", "client_id is required")
	}
	client, err := deps.ClientRepo.FindByID(ctx, tenancy.TenantID(ctx), clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, NewOAuthError("invalid_request", "The post_logout_redirect_uri is not registered.")
	}
	// post_logout_redirect_uri も認可の redirect_uri と同じ完全一致規則で照合する。
	if !domain.RedirectURIAllowed(client.RedirectURIs, in.PostLogoutRedirectURI) {
		return nil, NewOAuthError("invalid_request", "The post_logout_redirect_uri is not registered.")
	}
	target.Client = client
	target.RedirectURI = in.PostLogoutRedirectURI
	return target, nil
}

// checkSessionSubject は sid が指す LoginSession の主体が hint の sub と同じであることを
// 確かめる。セッションが無いときは失効させる対象が無いので拒否しない。
func checkSessionSubject(ctx context.Context, owners ports.LoginSessionOwnerLookup, sid, subject string) error {
	if owners == nil {
		return nil
	}
	owner, found, err := owners.LoginSessionOwner(ctx, sid)
	if err != nil {
		return err
	}
	if found && owner != subject {
		return NewOAuthError("invalid_request", "id_token_hint does not match the session subject")
	}
	return nil
}
