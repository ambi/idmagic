package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

const MaxActiveClientSecrets = 2

var (
	ErrClientSecretNotManageable       = errors.New("client secret is not manageable")
	ErrClientSecretLimitExceeded       = errors.New("active client secret limit exceeded")
	ErrClientSecretCredentialNotFound  = errors.New("client secret credential not found")
	ErrClientSecretCredentialNotActive = errors.New("client secret credential is not active")
)

type IssueClientSecretInput struct {
	ActorUserID   string
	ClientID      string
	ExpiresInDays int
	Now           time.Time
}

type IssueClientSecretResult struct {
	ClientSecret string
	Credential   domain.ClientSecretCredential
	Credentials  []domain.ClientSecretCredential
}

func IssueClientSecret(ctx context.Context, deps AdminOAuth2ClientDeps, in IssueClientSecretInput) (*IssueClientSecretResult, error) {
	if in.ExpiresInDays < 1 || in.ExpiresInDays > 730 {
		return nil, NewOAuthError("invalid_request", "expires_in_days must be between 1 and 730")
	}
	now := adminNow(in.Now)
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}
	if !clientSecretManageable(client) {
		return nil, ErrClientSecretNotManageable
	}
	var legacy *domain.ClientSecretCredential
	if client.ClientSecretHash != nil {
		legacyID, idErr := spec.NewUUIDv4()
		if idErr != nil {
			return nil, idErr
		}
		legacy = &domain.ClientSecretCredential{
			CredentialID: legacyID, ClientID: client.ClientID,
			SecretHash: *client.ClientSecretHash, CreatedAt: client.CreatedAt,
		}
	}
	secret, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	credentialID, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	expiresAt := now.AddDate(0, 0, in.ExpiresInDays)
	credential := domain.ClientSecretCredential{
		CredentialID: credentialID, ClientID: client.ClientID,
		SecretHash: domain.HashClientSecret(secret), CreatedAt: now, ExpiresAt: &expiresAt,
	}
	if err := deps.ClientRepo.IssueClientSecretCredential(ctx, legacy, credential, MaxActiveClientSecrets, now); err != nil {
		if errors.Is(err, oauthports.ErrClientSecretCredentialLimitExceeded) {
			return nil, ErrClientSecretLimitExceeded
		}
		return nil, err
	}
	credentials, err := deps.ClientRepo.ListClientSecretCredentials(ctx, client.ClientID)
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.ClientSecretIssued{
		At: now, TenantID: tenantID, ActorUserID: in.ActorUserID,
		ClientID: client.ClientID, CredentialID: credential.CredentialID, ExpiresAt: expiresAt,
	})
	return &IssueClientSecretResult{ClientSecret: secret, Credential: credential, Credentials: credentials}, nil
}

type RevokeClientSecretInput struct {
	ActorUserID  string
	ClientID     string
	CredentialID string
	Now          time.Time
}

type RevokeClientSecretResult struct {
	Credentials []domain.ClientSecretCredential
}

func RevokeClientSecret(ctx context.Context, deps AdminOAuth2ClientDeps, in RevokeClientSecretInput) (*RevokeClientSecretResult, error) {
	now := adminNow(in.Now)
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}
	if !clientSecretManageable(client) {
		return nil, ErrClientSecretNotManageable
	}
	credentials, err := deps.ClientRepo.ListClientSecretCredentials(ctx, client.ClientID)
	if err != nil {
		return nil, err
	}
	for i := range credentials {
		if credentials[i].CredentialID != in.CredentialID {
			continue
		}
		if credentials[i].RevokedAt != nil {
			return &RevokeClientSecretResult{Credentials: credentials}, nil
		}
		if !credentials[i].IsActiveAt(now) {
			return nil, ErrClientSecretCredentialNotActive
		}
		credentials[i].RevokedAt = &now
		if err := deps.ClientRepo.UpdateClientSecretCredential(ctx, credentials[i]); err != nil {
			return nil, err
		}
		emit(deps.Emit, &domain.ClientSecretRevoked{
			At: now, TenantID: tenantID, ActorUserID: in.ActorUserID,
			ClientID: client.ClientID, CredentialID: credentials[i].CredentialID,
		})
		return &RevokeClientSecretResult{Credentials: credentials}, nil
	}
	return nil, ErrClientSecretCredentialNotFound
}

func clientSecretManageable(client *domain.OAuth2Client) bool {
	return client.TokenEndpointAuthMethod == domain.AuthMethodClientSecretBasic ||
		client.TokenEndpointAuthMethod == domain.AuthMethodClientSecretPost
}
