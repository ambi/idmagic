package ports

import (
	"context"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
)

type Repository interface {
	Save(ctx context.Context, token *apitokendomain.ApiToken) error
	FindByJTI(ctx context.Context, tenantID, jti string) (*apitokendomain.ApiToken, error)
	List(ctx context.Context, tenantID string) ([]*apitokendomain.ApiToken, error)
	Revoke(ctx context.Context, tenantID, id string, at time.Time) error
	RevokeByJTI(ctx context.Context, tenantID, jti string, at time.Time) error
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (apitokendomain.Principal, error)
}
