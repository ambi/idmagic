package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
)

type Repository interface {
	Save(ctx context.Context, token *domain.ApiToken) error
	FindByJTI(ctx context.Context, tenantID, jti string) (*domain.ApiToken, error)
	List(ctx context.Context, tenantID string) ([]*domain.ApiToken, error)
	Revoke(ctx context.Context, tenantID, id string, at time.Time) error
	RevokeByJTI(ctx context.Context, tenantID, jti string, at time.Time) error
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (domain.Principal, error)
}
