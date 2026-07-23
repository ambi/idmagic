package usecases

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// =====================================================================
// /login (POST) → 認証完了 → authorization code 発行
// =====================================================================

type CompleteLoginDeps struct {
	RequestStore ports.AuthorizationRequestStore
	CodeStore    ports.AuthorizationCodeStore
}

type CompleteLoginInput struct {
	RequestID string
	Sub       string
	AuthTime  time.Time
	AMR       []string
	ACR       string
	// Sid は AuthenticationContext.session_id から一度だけ伝搬する OIDC session id
	// (ADR-127)。空文字列は browser session を持たない発行を表す。
	Sid string
}

type CompleteLoginOutput struct {
	Request *domain.AuthorizationRequest
	Code    *domain.AuthorizationCodeRecord
}

// CompleteLogin は認証・同意確認済みのリクエストに対して状態機械を回し、
// 認可コードを発行する。同意の要否判断と保存は HTTP 継続処理が先に行う。
func CompleteLogin(ctx context.Context, deps CompleteLoginDeps, in CompleteLoginInput) (*CompleteLoginOutput, error) {
	req, err := deps.RequestStore.Find(ctx, in.RequestID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, NewOAuthError("invalid_request", "unknown authorization request")
	}
	if req.TenantID != tenancy.TenantID(ctx) {
		return nil, NewOAuthError("invalid_request", "unknown authorization request")
	}
	if time.Now().After(req.ExpiresAt) {
		_ = deps.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowExpired)
		return nil, NewOAuthError("invalid_request", "authorization request expired")
	}
	if req.State != spec.AuthFlowReceived {
		return nil, NewOAuthError(
			"invalid_request",
			"The authorization request has already been processed. Restart authorization from the client.",
		)
	}

	// received → authentication_pending → authenticated → code_issued の最短経路
	if err := deps.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowAuthenticationPending); err != nil {
		return nil, err
	}
	if err := deps.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowAuthenticated); err != nil {
		return nil, err
	}
	authTime := in.AuthTime.UTC().Unix()
	if err := deps.RequestStore.AttachAuthentication(ctx, req.ID, in.Sub, authTime, in.AMR, in.ACR, in.Sid); err != nil {
		return nil, err
	}
	if err := deps.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowCodeIssued); err != nil {
		return nil, err
	}

	codeValue, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	record := &domain.AuthorizationCodeRecord{
		Code:                   codeValue,
		TenantID:               req.TenantID,
		AuthorizationRequestID: req.ID,
		ClientID:               req.ClientID,
		UserID:                 in.Sub,
		Scopes:                 strings.Fields(req.Scope),
		RedirectURI:            req.RedirectURI,
		CodeChallenge:          req.CodeChallenge,
		CodeChallengeMethod:    req.CodeChallengeMethod,
		Nonce:                  req.Nonce,
		AuthTime:               authTime,
		AMR:                    slices.Clone(in.AMR),
		ACR:                    optional(in.ACR),
		Sid:                    optional(in.Sid),
		AuthorizationDetails:   req.AuthorizationDetails,
		Resource:               req.Resource,
		State:                  spec.AuthCodeRecordIssued,
		IssuedAt:               now,
		ExpiresAt:              now.Add(60 * time.Second), // RFC 9700 §4.10
	}
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if err := deps.CodeStore.Save(ctx, record); err != nil {
		return nil, err
	}
	return &CompleteLoginOutput{Request: req, Code: record}, nil
}
