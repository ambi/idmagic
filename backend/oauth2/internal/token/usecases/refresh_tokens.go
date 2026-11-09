// リフレッシュトークンによる再発行。ローテーション + ファミリー失効。
package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"

	"github.com/ambi/idmagic/backend/shared/security/entropy"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type RefreshInput struct {
	ClientID     string
	RefreshToken string
	Scope        string
	Resource     []string
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
	ClientRepo            ports.OAuth2ClientRepository
	UserRepo              userports.UserRepository
	RefreshStore          ports.RefreshTokenStore
	McpResourceServerRepo ports.McpResourceServerRepository
	// UserEpochs は User の失効エポックより前の発行を拒む (REQ-SHAREDSIGNALS-012)。nil なら判定しない。
	UserEpochs  ssports.UserRevocationEpochRepository
	TokenIssuer ports.TokenIssuer
	Authorizer  policyports.Authorizer
	Emit        func(spec.DomainEvent)
}

func RefreshTokens(ctx context.Context, deps RefreshDeps, in RefreshInput, now time.Time) (*RefreshResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tenantID := tenantports.TenantID(ctx)
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
		emit(deps.Emit, &domain.RefreshTokenReuseDetected{At: now, TenantID: tenantID, FamilyID: record.FamilyID, TokenID: record.ID, ClientID: client.ClientID, UserID: record.UserID})
		return nil, NewOAuthError("invalid_grant", "The refresh token owner does not match.")
	}
	if domain.IsRefreshTokenReplay(record) {
		revokeFamilyAndNotify(ctx, deps.RefreshStore, deps.Emit, record.FamilyID, tenantID, now)
		emit(deps.Emit, &domain.RefreshTokenReuseDetected{At: now, TenantID: tenantID, FamilyID: record.FamilyID, TokenID: record.ID, ClientID: client.ClientID, UserID: record.UserID})
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
	if deps.UserEpochs != nil {
		epoch, err := deps.UserEpochs.FindByUser(ctx, tenantID, record.UserID)
		if err != nil {
			return nil, err
		}
		if epoch != nil && epoch.Supersedes(record.IssuedAt) {
			_, _ = deps.RefreshStore.RevokeFamily(ctx, record.FamilyID)
			return nil, NewOAuthError("invalid_grant", "The refresh token was revoked.")
		}
	}

	d, err := evaluateRefreshPolicy(ctx, deps.Authorizer, client, record, in, now)
	if err != nil {
		return nil, err
	}
	if !d.Permit {
		return nil, NewOAuthError("invalid_grant", "Refresh was rejected: "+strings.Join(d.Reasons, ", "))
	}

	scopes := record.Scopes
	if requested := strings.Fields(in.Scope); len(requested) > 0 {
		for _, scope := range requested {
			if !slices.Contains(record.Scopes, scope) {
				return nil, NewOAuthError("invalid_scope", "The requested scope exceeds the original grant.")
			}
		}
		scopes = requested
	}
	if err := validateRefreshResource(ctx, deps.McpResourceServerRepo, tenantID, in.Resource, record.Resource, scopes); err != nil {
		if oe, ok := errors.AsType[*OAuthError](err); ok && oe.Code == "invalid_target" {
			emit(deps.Emit, &domain.ResourceAudienceRejected{At: now, TenantID: tenantID, ClientID: client.ClientID, Reason: "invalid_target"})
		}
		return nil, err
	}
	var audiences []string
	if record.Resource != nil {
		audiences = []string{*record.Resource}
	}

	// ローテーションの確定後に拒否すると、再試行が再利用と判定されファミリーごと失効する (REQ-OAUTH2-075)。
	if guard, ok := deps.TokenIssuer.(ports.IssuanceGuard); ok {
		if err := guard.Precheck(ctx, ports.AccessTokenInput{Client: client, Sub: record.UserID, Scopes: scopes, Audiences: audiences}); err != nil {
			return nil, err
		}
	}

	newTok, err := domain.RotateRefreshToken(record, now, entropy.Crypto())
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

	// RFC 6749 §6: アクセストークンだけを絞り、次のリフレッシュトークンには元の付与を残す。
	access, jti, err := deps.TokenIssuer.SignAccessToken(ctx, ports.AccessTokenInput{
		Client:           client,
		Sub:              record.UserID,
		Scopes:           scopes,
		SenderConstraint: record.SenderConstraint,
		AuthTime:         now.Unix(),
		Audiences:        audiences,
	})
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AccessTokenIssued{At: now, TenantID: tenantID, JTI: jti, ClientID: client.ClientID, UserID: record.UserID, Scopes: scopes, SenderConstraint: senderConstraintTag(record.SenderConstraint)})
	if newTok.Record.Resource != nil {
		emit(deps.Emit, &domain.ResourceScopedTokenIssued{At: now, TenantID: tenantID, ClientID: client.ClientID, Resource: *newTok.Record.Resource, Scopes: scopes})
	}

	tokenType := domain.PresentationTokenType(record.SenderConstraint)
	return &RefreshResult{
		AccessToken:  access,
		RefreshToken: newTok.Token,
		TokenType:    tokenType,
		ExpiresIn:    deps.TokenIssuer.AccessTokenTTLSeconds(),
		Scope:        strings.Join(scopes, " "),
	}, nil
}

func validateRefreshResource(ctx context.Context, repo ports.McpResourceServerRepository, tenantID string, requested []string, bound *string, scopes []string) error {
	if len(requested) > 1 {
		return NewOAuthError("invalid_target", "only one resource can be specified (1 token = 1 resource)")
	}
	if len(requested) == 0 || requested[0] == "" {
		return nil
	}
	if bound == nil || requested[0] != *bound {
		return NewOAuthError("invalid_target", "The requested resource does not match the original grant.")
	}
	_, err := ResolveResourceIndicator(ctx, repo, tenantID, requested, scopes)
	return err
}

func evaluateRefreshPolicy(
	ctx context.Context,
	authorizer policyports.Authorizer,
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
