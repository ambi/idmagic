package usecases

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/apitoken/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var (
	ErrInvalidRequest = errors.New("invalid API token request")
	ErrAccessDenied   = errors.New("API token access denied")
)

type Service struct {
	repository ports.Repository
	now        func() time.Time
	random     io.Reader
}

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(service *Service) { service.now = now }
}

func WithRandomReader(random io.Reader) Option {
	return func(service *Service) { service.random = random }
}

func New(repository ports.Repository, options ...Option) *Service {
	service := &Service{repository: repository, now: time.Now, random: cryptorand.Reader}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) Issue(
	ctx context.Context,
	tenantID string,
	description string,
	scopeValues []string,
	expiryDays int,
) (string, domain.Metadata, error) {
	if expiryDays <= 0 {
		return "", domain.Metadata{}, ErrInvalidRequest
	}
	scopes, err := domain.ParseScopes(scopeValues)
	if err != nil {
		return "", domain.Metadata{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(s.random, raw); err != nil {
		return "", domain.Metadata{}, fmt.Errorf("generate API token: %w", err)
	}
	literalValue := domain.TokenPrefix + hex.EncodeToString(raw)
	literal, err := domain.ParseTokenLiteral(literalValue)
	if err != nil {
		return "", domain.Metadata{}, fmt.Errorf("generate API token: %w", err)
	}
	createdAt := s.now()
	expiresAt := createdAt.AddDate(0, 0, expiryDays)
	id, err := spec.NewUUIDv4()
	if err != nil {
		return "", domain.Metadata{}, fmt.Errorf("generate API token id: %w", err)
	}
	token := &domain.ApiToken{
		ID:          id,
		TenantID:    tenantID,
		TokenHash:   literal.Hash(),
		Scopes:      scopes,
		Description: description,
		CreatedAt:   createdAt,
		ExpiresAt:   &expiresAt,
	}
	if err := s.repository.Save(ctx, token); err != nil {
		return "", domain.Metadata{}, err
	}
	return literalValue, token.Metadata(), nil
}

func (s *Service) Authenticate(ctx context.Context, literalValue string) (domain.Principal, error) {
	literal, err := domain.ParseTokenLiteral(literalValue)
	if err != nil {
		return domain.Principal{}, ErrAccessDenied
	}
	token, err := s.repository.FindByHash(ctx, literal.Hash())
	if err != nil {
		return domain.Principal{}, err
	}
	if token == nil || len(token.Scopes) == 0 || (token.ExpiresAt != nil && !token.ExpiresAt.After(s.now())) {
		return domain.Principal{}, ErrAccessDenied
	}
	return domain.Principal{TenantID: token.TenantID, Scopes: append(domain.Scopes(nil), token.Scopes...)}, nil
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
	return s.repository.Delete(ctx, tenantID, id)
}
