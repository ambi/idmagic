// Package ports defines the boundaries required by OAuth2 consent use cases.
package ports

import (
	"context"

	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
)

type ConsentRepository interface {
	Find(ctx context.Context, tenantID, sub, clientID string) (*consentdomain.Consent, error)
	FindAll(ctx context.Context, tenantID string) ([]*consentdomain.Consent, error)
	// ListPage returns up to limit consents for tenantID ordered by
	// (user_id, client_id) ascending, strictly after the keyset (afterUserID,
	// afterClientID). Pass "", "" for the first page. Backs ListAdminConsents
	// keyset pagination (wi-159).
	ListPage(ctx context.Context, tenantID, afterUserID, afterClientID string, limit int) ([]*consentdomain.Consent, error)
	ListPageBefore(ctx context.Context, tenantID, beforeUserID, beforeClientID string, limit int) ([]*consentdomain.Consent, error)
	Save(ctx context.Context, tenantID string, c *consentdomain.Consent) error
	Revoke(ctx context.Context, tenantID, sub, clientID string) error
	// DeleteAllForSub は anonymize cascade から呼ばれる。
	// 対象 sub の Consent を物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
