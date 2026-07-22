package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/apitoken/domain"
)

type Repository interface {
	Save(ctx context.Context, token *domain.ApiToken) error
	FindByHash(ctx context.Context, tokenHash string) (*domain.ApiToken, error)
	List(ctx context.Context, tenantID string) ([]*domain.ApiToken, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (domain.Principal, error)
}
