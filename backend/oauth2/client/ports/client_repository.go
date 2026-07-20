// Package ports: OAuth2 ユースケースが要求する境界。
package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

type OAuth2ClientRepository interface {
	FindByID(ctx context.Context, tenantID, clientID string) (*domain.OAuth2Client, error)
	Save(ctx context.Context, c *domain.OAuth2Client) error
	Delete(ctx context.Context, tenantID, clientID string) error
	FindAll(ctx context.Context, tenantID string) ([]*domain.OAuth2Client, error)
	ListClientSecretCredentials(ctx context.Context, clientID string) ([]domain.ClientSecretCredential, error)
	SaveClientSecretCredential(ctx context.Context, credential domain.ClientSecretCredential) error
	UpdateClientSecretCredential(ctx context.Context, credential domain.ClientSecretCredential) error
}
