// Package ports defines the boundaries required by OAuth2 consent use cases.
package ports

import (
	"context"

	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
)

type ConsentRepository interface {
	Find(ctx context.Context, tenantID, sub, clientID string) (*consentdomain.Consent, error)
	FindAll(ctx context.Context, tenantID string) ([]*consentdomain.Consent, error)
	Save(ctx context.Context, tenantID string, c *consentdomain.Consent) error
	Revoke(ctx context.Context, tenantID, sub, clientID string) error
	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	// 対象 sub の Consent を物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
