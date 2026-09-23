// リフレッシュトークンによる再発行。ローテーション + ファミリー失効。
package usecases

import (
	"context"
	"strings"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type RefreshInput struct {
	ClientID     string
	RefreshToken string
	ProofJKT     string
	ProofX5TS256 string
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Scope        string
}

type RefreshDeps struct {
	ClientRepo   ports.OAuth2ClientRepository
	UserRepo     userports.UserRepository
	RefreshStore ports.RefreshTokenStore
	TokenIssuer  ports.TokenIssuer
	Authorizer   ports.Authorizer
	Emit         func(spec.DomainEvent)
}

func RefreshTokens(ctx context.Context, deps RefreshDeps, in RefreshInput, now time.Time) (*RefreshResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, NewOAuthError("invalid_client", "Client authentication failed.")
	}
	hash := domain.HashRefreshToken(in.RefreshToken)
	record, err := deps.RefreshStore.FindByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, NewOAuthError("invalid_grant", "The refresh token is invalid.")
	}
	if record.TenantID != tenantID {
		return nil, NewOAuthError("invalid_grant", "The refresh token is invalid.")
	}
	if record.ClientID != client.ClientID {
		_, _ = deps.RefreshStore.RevokeFamily(ctx, record.FamilyID)
		emit(deps.Emit, &domain.RefreshTokenReuseDetected{At: now, TenantID: tenantID, FamilyID: record.FamilyID, TokenID: record.ID, ClientID: client.ClientID})
		return nil, NewOAuthError("invalid_grant", "The refresh token owner does not match.")
	}
	if domain.IsRefreshTokenReplay(record) {
		revokeFamilyAndNotify(ctx, deps.RefreshStore, deps.Emit, record.FamilyID, tenantID, now)
		emit(deps.Emit, &domain.RefreshTokenReuseDetected{At: now, TenantID: tenantID, FamilyID: record.FamilyID, TokenID: record.ID, ClientID: client.ClientID})
		return nil, NewOAuthError("invalid_grant", "The refresh token has already been used.")
	}
	if domain.IsRefreshTokenAbsoluteExpired(record, now) {
		return nil, NewOAuthError("invalid_grant", "The refresh token has exceeded its absolute lifetime.")
	}
	user, err := deps.UserRepo.FindBySub(ctx, record.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		_, _ = deps.RefreshStore.RevokeFamily(ctx, record.FamilyID)
		return nil, NewOAuthError("invalid_grant", "user is unavailable")
	}
	if user.TenantID != tenantID {
		return nil, NewOAuthError("invalid_grant", "The refresh token is invalid.")
	}
	if !user.IsActive() {
		_, _ = deps.RefreshStore.RevokeFamily(ctx, record.FamilyID)
		return nil, NewOAuthError("invalid_grant", "user is disabled")
	}

	d, err := evaluateRefreshPolicy(ctx, deps.Authorizer, client, record, in, now)
	if err != nil {
		return nil, err
	}
	if !d.Permit {
		return nil, NewOAuthError("invalid_grant", "Refresh was rejected: "+strings.Join(d.Reasons, ", "))
	}

	newTok, err := domain.RotateRefreshToken(record, now)
	if err != nil {
		return nil, err
	}
	rotated, err := deps.RefreshStore.Rotate(ctx, record.ID, newTok.Record)
	if err != nil {
		return nil, err
	}
	if rotated == nil {
		_, _ = deps.RefreshStore.RevokeFamily(ctx, record.FamilyID)
		return nil, NewOAuthError("invalid_grant", "The refresh token was invalidated by a concurrent refresh.")
	}
	emit(deps.Emit, &domain.RefreshTokenRotated{At: now, TenantID: tenantID, OldTokenID: record.ID, NewTokenID: newTok.Record.ID, FamilyID: record.FamilyID})

	// resource indicator (wi-262): 初回発行時に束縛された resource を
	// rotation を跨いで保持する。RotateRefreshToken が parent.Resource を
	// 新レコードへ引き継ぎ済みなので、ここでは newTok.Record.Resource を使う。
	var audiences []string
	if newTok.Record.Resource != nil {
		audiences = []string{*newTok.Record.Resource}
	}
	access, jti, err := deps.TokenIssuer.SignAccessToken(ctx, ports.AccessTokenInput{
		Client:           client,
		Sub:              record.UserID,
		Scopes:           record.Scopes,
		SenderConstraint: record.SenderConstraint,
		AuthTime:         now.Unix(),
		Audiences:        audiences,
	})
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AccessTokenIssued{At: now, TenantID: tenantID, JTI: jti, ClientID: client.ClientID, UserID: record.UserID, Scopes: record.Scopes, SenderConstraint: senderConstraintTag(record.SenderConstraint)})
	if newTok.Record.Resource != nil {
		emit(deps.Emit, &domain.ResourceScopedTokenIssued{At: now, TenantID: tenantID, ClientID: client.ClientID, Resource: *newTok.Record.Resource, Scopes: record.Scopes})
	}

	tokenType := domain.PresentationTokenType(record.SenderConstraint)
	return &RefreshResult{
		AccessToken:  access,
		RefreshToken: newTok.Token,
		TokenType:    tokenType,
		ExpiresIn:    deps.TokenIssuer.AccessTokenTTLSeconds(),
		Scope:        strings.Join(record.Scopes, " "),
	}, nil
}

func evaluateRefreshPolicy(
	ctx context.Context,
	authorizer ports.Authorizer,
	client *domain.OAuth2Client,
	record *domain.RefreshTokenRecord,
	in RefreshInput,
	now time.Time,
) (spec.AuthZResponse, error) {
	props := spec.AuthZResourceProps{
		Revoked:           record.Revoked,
		Rotated:           record.Rotated,
		AbsoluteExpiresAt: record.AbsoluteExpiresAt,
		SenderConstraint:  authZSenderConstraint(record.SenderConstraint),
	}
	var pop *spec.AuthZProofOfPossession
	if record.SenderConstraint != nil {
		valid := false
		switch record.SenderConstraint.Type {
		case spec.SenderConstraintDPoP:
			valid = record.SenderConstraint.JKT == in.ProofJKT
		case spec.SenderConstraintMTLS:
			valid = record.SenderConstraint.X5TS256 == in.ProofX5TS256
		}
		pop = &spec.AuthZProofOfPossession{Valid: valid}
	}
	req := spec.AuthZRequest{
		Subject:  spec.AuthZSubject{Type: "Client", ID: client.ClientID, Properties: spec.AuthZSubjectProps{GrantTypes: client.GrantTypes}},
		Action:   spec.ActionTokenGrantRefresh,
		Resource: spec.AuthZResource{Type: "RefreshToken", Properties: props},
		Context:  spec.AuthZContext{ProofOfPossession: pop, Now: now},
	}
	if authorizer == nil {
		return spec.Evaluate(req), nil
	}
	return authorizer.Authorize(ctx, req)
}

// revokeFamilyAndNotify は再利用検出で発行ファミリーを失効させ、実際に Revoked へ
// 遷移した token ごとに TokenRevoked を発行する (RFC 9700 §4.10)。認可コード再提示
// (exchange_code.go の revokeReplayedFamily) と refresh トークン再利用 (このファイル) は
// 同じ状況を表すので、通知の組み立てを 1 か所にまとめて両者が同じ形で黙らないようにする。
func revokeFamilyAndNotify(ctx context.Context, store ports.RefreshTokenStore, emitFn func(spec.DomainEvent), familyID, tenantID string, now time.Time) {
	if store == nil {
		return
	}
	revokedIDs, _ := store.RevokeFamily(ctx, familyID)
	for _, tokenID := range revokedIDs {
		emit(emitFn, &domain.TokenRevoked{At: now, TenantID: tenantID, TokenType: "refresh_token", TokenID: tokenID, Reason: "reuse_detected"})
	}
}

func authZSenderConstraint(sc *domain.SenderConstraint) *spec.AuthZSenderConstraint {
	if sc == nil {
		return nil
	}
	return &spec.AuthZSenderConstraint{Type: sc.Type}
}
