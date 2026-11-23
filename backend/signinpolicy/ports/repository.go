package ports

import (
	"context"

	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
)

// SignInPolicyRepository は Application sign-in policy の永続境界。
type SignInPolicyRepository interface {
	// Get は application_id に一致する policy を返す。未設定なら (nil, nil)。
	Get(ctx context.Context, tenantID, applicationID string) (*policydomain.AppSignInPolicy, error)
	// ListAll はテナント内のすべての Application sign-in policy を返す。
	ListAll(ctx context.Context, tenantID string) ([]*policydomain.AppSignInPolicy, error)
	// Save は policy を upsert する。
	Save(ctx context.Context, policy *policydomain.AppSignInPolicy) error
	Update(ctx context.Context, tenantID, applicationID string, change func(*policydomain.AppSignInPolicy) (*policydomain.AppSignInPolicy, error)) (*policydomain.AppSignInPolicy, error)
	// Delete は application_id に一致する policy を削除する (冪等)。
	Delete(ctx context.Context, tenantID, applicationID string) error
}

// DefaultSignInPolicyRepository はテナント既定 sign-in policy の永続境界。
type DefaultSignInPolicyRepository interface {
	// Get は tenant_id に一致する既定 policy を返す。未設定なら (nil, nil)。
	Get(ctx context.Context, tenantID string) (*policydomain.TenantDefaultSignInPolicy, error)
	// Save は既定 policy を upsert する。
	Save(ctx context.Context, policy *policydomain.TenantDefaultSignInPolicy) error
	Update(ctx context.Context, tenantID string, change func(*policydomain.TenantDefaultSignInPolicy) (*policydomain.TenantDefaultSignInPolicy, error)) (*policydomain.TenantDefaultSignInPolicy, error)
}
