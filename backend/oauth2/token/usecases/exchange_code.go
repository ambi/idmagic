package usecases

import (
	"context"
	"slices"
	"strings"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/logout/domain"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// =====================================================================
// /token (authorization_code grant) → access_token + id_token
// =====================================================================

type ExchangeCodeDeps struct {
	ClientRepo         ports.OAuth2ClientRepository
	UserRepo           userports.UserRepository
	RequestStore       ports.AuthorizationRequestStore
	CodeStore          ports.AuthorizationCodeStore
	RefreshStore       ports.RefreshTokenStore
	TokenIssuer        ports.TokenIssuer
	ClientSessionStore logoutports.ClientSessionStore
	Emit               func(spec.DomainEvent)
	// ResolveAttributeDefs は ID Token の属性 claim 生成用 (wi-19)。nil 可。
	ResolveAttributeDefs func(ctx context.Context, tenantID string) ([]userdomain.UserAttributeDef, error)
}

type ExchangeCodeInput struct {
	ClientID     string
	Code         string
	CodeVerifier string
	RedirectURI  string
	DpopJKT      string
	MTLSX5TS256  string
	// Resource は RFC 8707 resource indicator。/token へ再指定された場合、
	// /authorize 時に束縛された resource と一致しなければならない (RFC 8707 §2)。
	Resource []string
}

type ExchangeCodeOutput struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Scope        string
}

func ExchangeCodeForToken(ctx context.Context, deps ExchangeCodeDeps, in ExchangeCodeInput) (*ExchangeCodeOutput, error) {
	if in.Code == "" {
		return nil, NewOAuthError("invalid_request", "code is required")
	}
	if in.CodeVerifier == "" {
		return nil, NewOAuthError("invalid_request", "code_verifier is required")
	}
	if in.RedirectURI == "" {
		return nil, NewOAuthError("invalid_request", "redirect_uri is required")
	}

	rec, err := deps.CodeStore.Find(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, NewOAuthError("invalid_grant", "The authorization code is invalid.")
	}
	tenantID := tenancy.TenantID(ctx)
	if rec.TenantID != tenantID {
		return nil, NewOAuthError("invalid_grant", "The authorization code is invalid.")
	}
	now := time.Now().UTC()
	if rec.State != spec.AuthCodeRecordIssued || !now.Before(rec.ExpiresAt) {
		revokeReplayedFamily(ctx, deps, rec, now, tenantID)
		return nil, NewOAuthError("invalid_grant", "The authorization code has been used or has expired.")
	}
	if rec.ClientID != in.ClientID {
		return nil, NewOAuthError("invalid_grant", "The authorization code is not bound to the client.")
	}
	if rec.RedirectURI != in.RedirectURI {
		return nil, NewOAuthError("invalid_grant", "The redirect_uri does not match.")
	}
	if !domain.VerifyPKCES256(in.CodeVerifier, rec.CodeChallenge) {
		return nil, NewOAuthError("invalid_grant", "PKCE verification failed")
	}
	// RFC 8707 §2 — /token に resource が再指定された場合、/authorize 時に束縛された
	// resource と一致しなければならない。新規 resource の後付け指定は拒否する。
	if requested := nonEmpty(in.Resource); len(requested) > 0 {
		if len(requested) > 1 || rec.Resource == nil || requested[0] != *rec.Resource {
			emit(deps.Emit, &domain.ResourceAudienceRejected{At: time.Now().UTC(), TenantID: tenantID, ClientID: in.ClientID, Reason: "invalid_target"})
			return nil, NewOAuthError("invalid_target", "token request resource does not match authorization request")
		}
	}

	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, NewOAuthError("invalid_client", "unknown client_id")
	}
	user, err := deps.UserRepo.FindBySub(ctx, rec.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewOAuthError("invalid_grant", "user is unavailable")
	}
	if user.TenantID != tenantID {
		return nil, NewOAuthError("invalid_grant", "The authorization code is invalid.")
	}
	if !user.IsActive() {
		return nil, NewOAuthError("invalid_grant", "user is disabled")
	}
	redeemed, err := deps.CodeStore.Redeem(ctx, in.Code, now)
	if err != nil {
		return nil, err
	}
	if redeemed == nil {
		revokeReplayedFamily(ctx, deps, rec, now, tenantID)
		return nil, NewOAuthError("invalid_grant", "The authorization code was used by a concurrent request.")
	}
	rec = redeemed

	var sc *domain.SenderConstraint
	if in.DpopJKT != "" {
		sc = &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: in.DpopJKT}
	} else if in.MTLSX5TS256 != "" {
		sc = &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: in.MTLSX5TS256}
	}

	var audiences []string
	if rec.Resource != nil {
		audiences = []string{*rec.Resource}
	}
	access, jti, err := deps.TokenIssuer.SignAccessToken(ctx, ports.AccessTokenInput{
		Client:               client,
		Sub:                  user.ID,
		Scopes:               rec.Scopes,
		SenderConstraint:     sc,
		AuthTime:             rec.AuthTime,
		AMR:                  rec.AMR,
		ACR:                  optionalValue(rec.ACR),
		AuthorizationDetails: rec.AuthorizationDetails,
		Audiences:            audiences,
	})
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AccessTokenIssued{At: now, TenantID: tenantID, JTI: jti, ClientID: client.ClientID, UserID: user.ID, Scopes: rec.Scopes, SenderConstraint: senderConstraintTag(sc)})
	emit(deps.Emit, &domain.AuthorizationCodeRedeemed{At: now, TenantID: tenantID, ClientID: client.ClientID, UserID: user.ID})
	if rec.Resource != nil {
		emit(deps.Emit, &domain.ResourceScopedTokenIssued{At: now, TenantID: tenantID, ClientID: client.ClientID, Resource: *rec.Resource, Scopes: rec.Scopes})
	}

	var idToken string
	if slices.Contains(rec.Scopes, "openid") {
		idToken, err = deps.TokenIssuer.SignIDToken(ctx, ports.IDTokenInput{
			Client:    client,
			User:      user,
			Scopes:    rec.Scopes,
			Nonce:     rec.Nonce,
			AuthTime:  rec.AuthTime,
			AMR:       rec.AMR,
			ACR:       optionalValue(rec.ACR),
			Sid:       optionalValue(rec.Sid),
			AtHashFor: access,

			ResolveAttributeDefs: deps.ResolveAttributeDefs,
		})
		if err != nil {
			return nil, err
		}
	}

	var refreshToken string
	if deps.RefreshStore != nil && slices.Contains(rec.Scopes, "offline_access") {
		gen, err := domain.GenerateInitialRefreshToken(client.ClientID, user.ID, rec.Scopes, sc, rec.Sid, rec.Resource, now)
		if err != nil {
			return nil, err
		}
		gen.Record.TenantID = tenantID
		if err := deps.RefreshStore.Save(ctx, gen.Record); err != nil {
			return nil, err
		}
		emit(deps.Emit, &domain.RefreshTokenIssued{At: now, TenantID: tenantID, TokenID: gen.Record.ID, FamilyID: gen.Record.FamilyID, ClientID: client.ClientID, UserID: user.ID})
		if err := deps.CodeStore.LinkFamily(ctx, rec.Code, gen.Record.FamilyID); err != nil {
			return nil, err
		}
		refreshToken = gen.Token
	}

	if deps.RequestStore != nil {
		_ = deps.RequestStore.UpdateState(ctx, rec.AuthorizationRequestID, spec.AuthFlowExchanged)
	}
	if deps.ClientSessionStore != nil && rec.Sid != nil && *rec.Sid != "" {
		if err := deps.ClientSessionStore.Upsert(ctx, &logoutdomain.ClientSession{
			TenantID: tenantID, Sid: *rec.Sid, ClientID: client.ClientID,
			FirstIssuedAt: now, LastIssuedAt: now,
		}); err != nil {
			return nil, err
		}
	}

	tokenType := domain.PresentationTokenType(sc)
	return &ExchangeCodeOutput{
		AccessToken:  access,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		ExpiresIn:    deps.TokenIssuer.AccessTokenTTLSeconds(),
		Scope:        strings.Join(rec.Scopes, " "),
	}, nil
}

func optionalValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// revokeReplayedFamily は、使用済みまたは期限切れの認可コードが再び提示されたときに
// 発行ファミリーを失効させ、検出を通知する (RFC 9700 §4.10、REQ-OAUTH2-005)。
//
// 通知は refresh トークンの再利用検出と同じ意味を持つ。RefreshToken を再提示する経路は
// refresh_tokens.go が同じ組で失効と通知を行っており、認可コードを再提示する経路だけが
// 失効はしても黙っていた。監査から見ると、片方の再利用は記録に残り、もう片方は残らない。
func revokeReplayedFamily(
	ctx context.Context, deps ExchangeCodeDeps, rec *domain.AuthorizationCodeRecord,
	now time.Time, tenantID string,
) {
	if rec.IssuedFamilyID == nil || deps.RefreshStore == nil {
		return
	}
	_ = deps.RefreshStore.RevokeFamily(ctx, *rec.IssuedFamilyID)
	emit(deps.Emit, &domain.RefreshTokenReuseDetected{
		At: now, TenantID: tenantID, FamilyID: *rec.IssuedFamilyID, ClientID: rec.ClientID,
	})
}
