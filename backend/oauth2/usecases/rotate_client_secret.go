package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

var ErrClientSecretNotRotatable = errors.New("client secret is not rotatable")

type RotateClientSecretInput struct {
	ActorUserID string
	ClientID    string
	GraceDays   int
	Now         time.Time
}

type RotateClientSecretResult struct {
	ClientSecret string
	GraceUntil   *time.Time
	Credentials  []domain.ClientSecretCredential
}

func RotateClientSecret(ctx context.Context, deps AdminOAuth2ClientDeps, in RotateClientSecretInput) (*RotateClientSecretResult, error) {
	if in.GraceDays < 0 || in.GraceDays > 30 {
		return nil, NewOAuthError("invalid_request", "grace_days must be between 0 and 30")
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
	if client.TokenEndpointAuthMethod != domain.AuthMethodClientSecretBasic && client.TokenEndpointAuthMethod != domain.AuthMethodClientSecretPost {
		return nil, ErrClientSecretNotRotatable
	}
	credentials, err := deps.ClientRepo.ListClientSecretCredentials(ctx, client.ClientID)
	if err != nil {
		return nil, err
	}
	// rollout 中に credential table が未 backfill の client は旧 hash を一度だけ移す。
	if len(credentials) == 0 && client.ClientSecretHash != nil {
		id, err := spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
		legacy := domain.ClientSecretCredential{CredentialID: id, ClientID: client.ClientID, SecretHash: *client.ClientSecretHash, CreatedAt: client.CreatedAt}
		if err := deps.ClientRepo.SaveClientSecretCredential(ctx, legacy); err != nil {
			return nil, err
		}
		credentials = append(credentials, legacy)
	}
	var graceUntil *time.Time
	if in.GraceDays > 0 {
		value := now.AddDate(0, 0, in.GraceDays)
		graceUntil = &value
	}
	for i := range credentials {
		if !credentials[i].IsActiveAt(now) {
			continue
		}
		if graceUntil == nil {
			credentials[i].RevokedAt = &now
		} else {
			credentials[i].ExpiresAt = graceUntil
		}
		if err := deps.ClientRepo.UpdateClientSecretCredential(ctx, credentials[i]); err != nil {
			return nil, err
		}
	}
	secret, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	credential := domain.ClientSecretCredential{CredentialID: id, ClientID: client.ClientID, SecretHash: domain.HashClientSecret(secret), CreatedAt: now}
	if err := deps.ClientRepo.SaveClientSecretCredential(ctx, credential); err != nil {
		return nil, err
	}
	credentials = append(credentials, credential)
	emit(deps.Emit, &domain.ClientSecretRotated{At: now, TenantID: tenantID, ActorUserID: in.ActorUserID, ClientID: client.ClientID, GraceUntil: graceUntil, RevokedImmediately: graceUntil == nil})
	return &RotateClientSecretResult{ClientSecret: secret, GraceUntil: graceUntil, Credentials: credentials}, nil
}
