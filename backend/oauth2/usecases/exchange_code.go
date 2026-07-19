package usecases

import (
	"context"
	"slices"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// =====================================================================
// /token (authorization_code grant) → access_token + id_token
// =====================================================================

type ExchangeCodeDeps struct {
	ClientRepo   ports.OAuth2ClientRepository
	UserRepo     idmports.UserRepository
	RequestStore ports.AuthorizationRequestStore
	CodeStore    ports.AuthorizationCodeStore
	RefreshStore ports.RefreshTokenStore
	TokenIssuer  ports.TokenIssuer
	Emit         func(spec.DomainEvent)
	// ResolveAttributeDefs は ID Token の属性 claim 生成用 (wi-19)。nil 可。
	ResolveAttributeDefs func(ctx context.Context, tenantID string) ([]idmdomain.UserAttributeDef, error)
}

type ExchangeCodeInput struct {
	ClientID     string
	Code         string
	CodeVerifier string
	RedirectURI  string
	DpopJKT      string
	MTLSX5TS256  string
	// Resource は RFC 8707 resource indicator (ADR-055)。/token へ再指定された場合、
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
		return nil, NewOAuthError("invalid_request", "code が必要です")
	}
	if in.CodeVerifier == "" {
		return nil, NewOAuthError("invalid_request", "code_verifier が必要です")
	}
	if in.RedirectURI == "" {
		return nil, NewOAuthError("invalid_request", "redirect_uri が必要です")
	}

	rec, err := deps.CodeStore.Find(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, NewOAuthError("invalid_grant", "code が無効です")
	}
	tenantID := tenancy.TenantID(ctx)
	if rec.TenantID != tenantID {
		return nil, NewOAuthError("invalid_grant", "code が無効です")
	}
	now := time.Now().UTC()
	if rec.State != spec.AuthCodeRecordIssued || !now.Before(rec.ExpiresAt) {
		if rec.IssuedFamilyID != nil && deps.RefreshStore != nil {
			_ = deps.RefreshStore.RevokeFamily(ctx, *rec.IssuedFamilyID)
		}
		return nil, NewOAuthError("invalid_grant", "code が使用済みまたは期限切れ")
	}
	if rec.ClientID != in.ClientID {
		return nil, NewOAuthError("invalid_grant", "code がクライアントに紐づかない")
	}
	if rec.RedirectURI != in.RedirectURI {
		return nil, NewOAuthError("invalid_grant", "redirect_uri が一致しない")
	}
	if !domain.VerifyPKCES256(in.CodeVerifier, rec.CodeChallenge) {
		return nil, NewOAuthError("invalid_grant", "PKCE 検証失敗")
	}
	// RFC 8707 §2 — /token に resource が再指定された場合、/authorize 時に束縛された
	// resource と一致しなければならない (ADR-055)。新規 resource の後付け指定は拒否する。
	if requested := nonEmpty(in.Resource); len(requested) > 0 {
		if len(requested) > 1 || rec.Resource == nil || requested[0] != *rec.Resource {
			emit(deps.Emit, &domain.ResourceAudienceRejected{At: time.Now().UTC(), TenantID: tenantID, ClientID: in.ClientID, Reason: "invalid_target"})
			return nil, NewOAuthError("invalid_target", "token リクエストの resource が認可リクエストと一致しません")
		}
	}

	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, NewOAuthError("invalid_client", "未知の client_id")
	}
	user, err := deps.UserRepo.FindBySub(ctx, rec.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewOAuthError("invalid_grant", "ユーザーは利用できません")
	}
	if user.TenantID != tenantID {
		return nil, NewOAuthError("invalid_grant", "code が無効です")
	}
	if !user.IsActive() {
		return nil, NewOAuthError("invalid_grant", "ユーザーは無効化されています")
	}
	redeemed, err := deps.CodeStore.Redeem(ctx, in.Code, now)
	if err != nil {
		return nil, err
	}
	if redeemed == nil {
		if rec.IssuedFamilyID != nil && deps.RefreshStore != nil {
			_ = deps.RefreshStore.RevokeFamily(ctx, *rec.IssuedFamilyID)
		}
		return nil, NewOAuthError("invalid_grant", "code は並行リクエストにより使用済みです")
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
		gen, err := domain.GenerateInitialRefreshToken(client.ClientID, user.ID, rec.Scopes, sc, rec.Sid, now)
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

	tokenType := "Bearer"
	if sc != nil && sc.Type == spec.SenderConstraintDPoP {
		tokenType = "DPoP"
	}
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
