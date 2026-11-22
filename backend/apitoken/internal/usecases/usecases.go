package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const BuiltinClientID = apitokendomain.BuiltinClientID

var (
	ErrInvalidRequest = errors.New("invalid API token request")
	ErrAccessDenied   = apitokendomain.ErrInvalidToken
)

type Service struct {
	repository   apitokenports.Repository
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

func New(repository apitokenports.Repository, options ...Option) *Service {
	service := &Service{repository: repository, now: time.Now}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) Authenticate(ctx context.Context, token string) (apitokendomain.Principal, error) {
	if s.introspector == nil {
		return apitokendomain.Principal{}, ErrAccessDenied
	}
	claims, err := s.introspector.IntrospectAccessToken(ctx, token)
	if err != nil || claims == nil {
		return apitokendomain.Principal{}, ErrAccessDenied
	}
	return s.AuthenticateClaims(ctx, tenantports.TenantID(ctx), *claims)
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
	if _, err := s.AuthenticateClaims(ctx, tenantports.TenantID(ctx), *claims); err != nil {
		return &oauthports.IntrospectionResult{Active: false}, nil //nolint:nilerr // RFC 7662 requires active:false without validation detail leakage.
	}
	return claims, nil
}

func (s *Service) Issue(ctx context.Context, tenantID, userID, description string, scopeValues []string, expiryDays int, dpopJKT string) (string, apitokendomain.Metadata, error) {
	if expiryDays <= 0 || strings.TrimSpace(userID) == "" || s.issuer == nil {
		return "", apitokendomain.Metadata{}, ErrInvalidRequest
	}
	scopes, err := apitokendomain.ParseScopes(scopeValues)
	if err != nil {
		return "", apitokendomain.Metadata{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}
	if len(scopes) == 0 {
		return "", apitokendomain.Metadata{}, ErrInvalidRequest
	}
	createdAt := s.now().UTC()
	expiresAt := createdAt.AddDate(0, 0, expiryDays)
	audience := tenantports.Issuer(ctx, "")
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
		return "", apitokendomain.Metadata{}, fmt.Errorf("sign API token: %w", err)
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return "", apitokendomain.Metadata{}, fmt.Errorf("generate API token id: %w", err)
	}
	token := &apitokendomain.ApiToken{
		ID: id, TenantID: tenantID, UserID: userID, JTI: jti, ClientID: BuiltinClientID,
		Scopes: scopes, Audience: audience, DPoPJKT: dpopJKT, Description: description, CreatedAt: createdAt, ExpiresAt: &expiresAt,
	}
	if err := s.repository.Save(ctx, token); err != nil {
		return "", apitokendomain.Metadata{}, err
	}
	return literal, token.Metadata(), nil
}

// IssuePersonal は本人向けの発行で権限と期限を制限し、署名と保存は共通の発行へ委ねる。
func (s *Service) IssuePersonal(ctx context.Context, tenantID, userID, description string, scopeValues []string, expiryDays int, dpopJKT string) (string, apitokendomain.Metadata, error) {
	if expiryDays < 1 || expiryDays > 365 || len(scopeValues) == 0 {
		return "", apitokendomain.Metadata{}, ErrInvalidRequest
	}
	for _, value := range scopeValues {
		if !spec.IsAccountScope(value) {
			return "", apitokendomain.Metadata{}, ErrInvalidRequest
		}
	}
	return s.Issue(ctx, tenantID, userID, description, scopeValues, expiryDays, dpopJKT)
}

func (s *Service) ListPersonal(ctx context.Context, tenantID, userID string) ([]apitokendomain.Metadata, error) {
	tokens, err := s.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]apitokendomain.Metadata, 0, len(tokens))
	for _, token := range tokens {
		if token.UserID == userID {
			result = append(result, token)
		}
	}
	return result, nil
}

func (s *Service) RevokePersonal(ctx context.Context, tenantID, userID, id string) error {
	tokens, err := s.ListPersonal(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		if token.ID == id {
			return s.Revoke(ctx, tenantID, id)
		}
	}
	return nil
}

func (s *Service) AuthenticateClaims(ctx context.Context, tenantID string, claims oauthports.IntrospectionResult) (apitokendomain.Principal, error) {
	if !claims.Active || !claims.Managed || claims.JTI == "" {
		return apitokendomain.Principal{}, ErrAccessDenied
	}
	token, err := s.repository.FindByJTI(ctx, tenantID, claims.JTI)
	if err != nil {
		return apitokendomain.Principal{}, err
	}
	now := s.now()
	if token == nil || token.RevokedAt != nil || len(token.Scopes) == 0 || token.ExpiresAt == nil || !token.ExpiresAt.After(now) {
		return apitokendomain.Principal{}, ErrAccessDenied
	}
	claimScopes, err := apitokendomain.ParseScopes(strings.Fields(claims.Scope))
	if err != nil || token.UserID != claims.Sub || token.ClientID != claims.ClientID ||
		!slices.Equal(token.Scopes, claimScopes) || len(claims.Aud) != 1 || claims.Aud[0] != token.Audience {
		return apitokendomain.Principal{}, ErrAccessDenied
	}
	return apitokendomain.Principal{
		TenantID: token.TenantID, UserID: token.UserID, ClientID: token.ClientID,
		Scopes: append(apitokendomain.Scopes(nil), token.Scopes...), Audience: token.Audience, TokenID: token.ID,
		IssuedAt: token.CreatedAt, ExpiresAt: token.ExpiresAt, DPoPJKT: token.DPoPJKT,
	}, nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]apitokendomain.Metadata, error) {
	tokens, err := s.repository.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]apitokendomain.Metadata, len(tokens))
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
