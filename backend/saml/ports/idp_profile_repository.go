package ports

import (
	"context"
	"time"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

type SamlIdentityProviderProfileRepository interface {
	// EnsureDefaultIDPProfile は既定のプロファイルを返す。なければ now に作成したものとして作る。
	EnsureDefaultIDPProfile(ctx context.Context, tenantID string, now time.Time) (*samldomain.SamlIdentityProviderProfile, error)
	FindIDPProfileByID(ctx context.Context, tenantID, profileID string) (*samldomain.SamlIdentityProviderProfile, error)
	// ListIDPProfilesByTenant は既定のプロファイルを EnsureDefaultIDPProfile と同じく now で補って返す。
	ListIDPProfilesByTenant(ctx context.Context, tenantID string, now time.Time) ([]*samldomain.SamlIdentityProviderProfile, error)
	// SaveIDPProfile は profile.UpdatedAt を更新の時刻として記録する。新しいプロファイルの作成の時刻は
	// profile.CreatedAt で、ゼロなら profile.UpdatedAt である。
	SaveIDPProfile(ctx context.Context, profile *samldomain.SamlIdentityProviderProfile) error
	DeleteIDPProfile(ctx context.Context, tenantID, profileID string) error
}
