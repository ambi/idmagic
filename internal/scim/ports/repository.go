package ports

import (
	"context"
	"time"
)

type ScimToken struct {
	ID          string
	TenantID    string
	TokenHash   string
	Description string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

type ScimUserRef struct {
	TenantID string
	ScimID   string
	UserID   string
}

type ScimGroupRef struct {
	TenantID string
	ScimID   string
	GroupID  string
}

type ScimRepository interface {
	SaveToken(ctx context.Context, token *ScimToken) error
	FindToken(ctx context.Context, tokenHash string) (*ScimToken, error)
	ListTokens(ctx context.Context, tenantID string) ([]*ScimToken, error)
	DeleteToken(ctx context.Context, tenantID, id string) error

	SaveUserRef(ctx context.Context, ref *ScimUserRef) error
	FindUserRefByScimID(ctx context.Context, tenantID, scimID string) (*ScimUserRef, error)
	FindUserRefByUserID(ctx context.Context, tenantID, userID string) (*ScimUserRef, error)
	DeleteUserRef(ctx context.Context, tenantID, scimID string) error

	SaveGroupRef(ctx context.Context, ref *ScimGroupRef) error
	FindGroupRefByScimID(ctx context.Context, tenantID, scimID string) (*ScimGroupRef, error)
	FindGroupRefByGroupID(ctx context.Context, tenantID, groupID string) (*ScimGroupRef, error)
	DeleteGroupRef(ctx context.Context, tenantID, scimID string) error
}
