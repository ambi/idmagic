// /par (RFC 9126 Pushed Authorization Request)
package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type PARInput struct {
	ClientID   string
	Parameters map[string]string
	// Resource は RFC 8707 resource indicator。form の resource は複数指定され得るため
	// Parameters (単一値に平坦化済み) とは別に slice で受ける (ADR-055)。
	Resource []string
}

type PARResult struct {
	RequestURI string
	ExpiresIn  int
}

type PARDeps struct {
	ClientRepo            ports.OAuth2ClientRepository
	Store                 ports.PARStore
	AuthzDetailTypeRepo   ports.AuthorizationDetailTypeRepository
	McpResourceServerRepo ports.McpResourceServerRepository
	Emit                  func(spec.DomainEvent)
}

func PushAuthorizationRequest(ctx context.Context, deps PARDeps, in PARInput, now time.Time) (*PARResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, NewOAuthError("invalid_client", "unknown client_id")
	}
	// RFC 9396 — authorization_details があれば push 時点で fail-closed 検証する (ADR-050)。
	if raw := in.Parameters["authorization_details"]; raw != "" {
		details, err := ParseAuthorizationDetails(raw)
		if err != nil {
			return nil, err
		}
		if err := ValidateAuthorizationDetails(ctx, deps.AuthzDetailTypeRepo, details); err != nil {
			return nil, err
		}
	}
	// RFC 8707 resource indicator — push 時点で早期に fail-closed 検証する (ADR-055)。
	// 確定検証は /authorize 消費時・/token 発行時にも再度行う (registry 状態変化への防御)。
	if _, err := ResolveResourceIndicator(ctx, deps.McpResourceServerRepo, tenantID, in.Resource, strings.Fields(in.Parameters["scope"])); err != nil {
		emit(deps.Emit, &domain.ResourceAudienceRejected{At: now, TenantID: tenantID, ClientID: in.ClientID, Reason: errorCode(err)})
		return nil, err
	}
	id, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	requestURI := "urn:ietf:params:oauth:request_uri:" + id
	rec := &domain.PARRecord{
		TenantID:   tenantID,
		RequestURI: requestURI,
		ClientID:   in.ClientID,
		Parameters: in.Parameters,
		IssuedAt:   now,
		ExpiresAt:  now.Add(90 * time.Second), // RFC 9126 §4 推奨上限
	}
	if err := rec.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Store.Save(ctx, rec); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.PARStored{At: now, TenantID: tenantID, RequestURI: requestURI, ClientID: in.ClientID})
	return &PARResult{RequestURI: requestURI, ExpiresIn: 90}, nil
}
