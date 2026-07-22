package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/apitoken/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

const BuiltinClientID = domain.BuiltinClientID

var (
	ErrInvalidRequest = errors.New("invalid API token request")
	ErrAccessDenied   = errors.New("API token access denied")
)

type Service struct {
	repository   ports.Repository
	issuer       oauthports.TokenIssuer
	introspector oauthports.TokenIntrospector
	now          func() time.Time
}

type Option func(*Service)

func WithClock(now func() time.Time) Option { return func(service *Service) { service.now = now } }
func WithTokenIssuer(issuer oauthports.TokenIssuer) Option {
	return func(service *Service) { service.issuer = issuer }
}

func WithTokenIntrospector(introspector oauthports.TokenIntrospector) Option {
	return func(service *Service) { service.introspector = introspector }
}

func New(repository ports.Repository, options ...Option) *Service {
	service := &Service{repository: repository, now: time.Now}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) Authenticate(ctx context.Context, token string) (domain.Principal, error) {
	if s.introspector == nil {
		return domain.Principal{}, ErrAccessDenied
	}
	claims, err := s.introspector.IntrospectAccessToken(ctx, token)
	if err != nil || claims == nil {
		return domain.Principal{}, ErrAccessDenied
	}
	return s.AuthenticateClaims(ctx, tenancy.TenantID(ctx), *claims)
}

// IntrospectAccessToken は共通 JWT 検証結果に managed-token lifecycle record の
// active 判定を重ねる。通常 OAuth JWT は共通検証結果をそのまま返す。
func (s *Service) IntrospectAccessToken(ctx context.Context, token string) (*oauthports.IntrospectionResult, error) {
	if s.introspector == nil {
		return &oauthports.IntrospectionResult{Active: false}, nil
	}
	claims, err := s.introspector.IntrospectAccessToken(ctx, token)
	if err != nil || claims == nil {
		return &oauthports.IntrospectionResult{Active: false}, err
	}
	if !claims.Active || !claims.Managed {
		return claims, nil
	}
	if _, err := s.AuthenticateClaims(ctx, tenancy.TenantID(ctx), *claims); err != nil {
		return &oauthports.IntrospectionResult{Active: false}, nil //nolint:nilerr // RFC 7662 requires active:false without validation detail leakage.
	}
	return claims, nil
}

func (s *Service) Issue(ctx context.Context, tenantID, userID, description string, scopeValues []string, expiryDays int, dpopJKT string) (string, domain.Metadata, error) {
	if expiryDays <= 0 || strings.TrimSpace(userID) == "" || s.issuer == nil {
		return "", domain.Metadata{}, ErrInvalidRequest
	}
	scopes, err := domain.ParseScopes(scopeValues)
	if err != nil {
		return "", domain.Metadata{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}
	if len(scopes) == 0 {
		return "", domain.Metadata{}, ErrInvalidRequest
	}
	createdAt := s.now().UTC()
	expiresAt := createdAt.AddDate(0, 0, expiryDays)
	audience := tenancy.Issuer(ctx, "")
	if audience == "" {
		audience = tenantID
	}
	client := &oauthdomain.OAuth2Client{TenantID: tenantID, ClientID: BuiltinClientID}
	var senderConstraint *oauthdomain.SenderConstraint
	if dpopJKT != "" {
		senderConstraint = &oauthdomain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: dpopJKT}
	}
	literal, jti, err := s.issuer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: client, Sub: userID, Scopes: scopes.Strings(), Audiences: []string{audience},
		SenderConstraint: senderConstraint, AuthTime: createdAt.Unix(), ExpiresAt: expiresAt.Unix(), Managed: true,
	})
	if err != nil {
		return "", domain.Metadata{}, fmt.Errorf("sign API token: %w", err)
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return "", domain.Metadata{}, fmt.Errorf("generate API token id: %w", err)
	}
	token := &domain.ApiToken{
		ID: id, TenantID: tenantID, UserID: userID, JTI: jti, ClientID: BuiltinClientID,
		Scopes: scopes, Audience: audience, DPoPJKT: dpopJKT, Description: description, CreatedAt: createdAt, ExpiresAt: &expiresAt,
	}
	if err := s.repository.Save(ctx, token); err != nil {
		return "", domain.Metadata{}, err
	}
	return literal, token.Metadata(), nil
}

func (s *Service) AuthenticateClaims(ctx context.Context, tenantID string, claims oauthports.IntrospectionResult) (domain.Principal, error) {
	if !claims.Active || !claims.Managed || claims.JTI == "" {
		return domain.Principal{}, ErrAccessDenied
	}
	token, err := s.repository.FindByJTI(ctx, tenantID, claims.JTI)
	if err != nil {
		return domain.Principal{}, err
	}
	now := s.now()
	if token == nil || token.RevokedAt != nil || len(token.Scopes) == 0 || token.ExpiresAt == nil || !token.ExpiresAt.After(now) {
		return domain.Principal{}, ErrAccessDenied
	}
	claimScopes, err := domain.ParseScopes(strings.Fields(claims.Scope))
	if err != nil || token.UserID != claims.Sub || token.ClientID != claims.ClientID ||
		!slices.Equal(token.Scopes, claimScopes) || len(claims.Aud) != 1 || claims.Aud[0] != token.Audience {
		return domain.Principal{}, ErrAccessDenied
	}
	return domain.Principal{
		TenantID: token.TenantID, UserID: token.UserID, ClientID: token.ClientID,
		Scopes: append(domain.Scopes(nil), token.Scopes...), Audience: token.Audience, TokenID: token.ID,
		IssuedAt: token.CreatedAt, ExpiresAt: token.ExpiresAt, DPoPJKT: token.DPoPJKT,
	}, nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]domain.Metadata, error) {
	tokens, err := s.repository.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Metadata, len(tokens))
	for i, token := range tokens {
		result[i] = token.Metadata()
	}
	return result, nil
}

func (s *Service) Revoke(ctx context.Context, tenantID, id string) error {
	return s.repository.Revoke(ctx, tenantID, id, s.now().UTC())
}

func (s *Service) RevokeByJTI(ctx context.Context, tenantID, jti string, at time.Time) error {
	return s.repository.RevokeByJTI(ctx, tenantID, jti, at.UTC())
}
