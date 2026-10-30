// Package ports defines the boundaries required by OAuth2 consent use cases.
package ports

import (
	"context"
	"time"

	consentdomain "github.com/ambi/idmagic/backend/oauth2/internal/consent/domain"
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

	// 以降は Agent への代行の付与の操作である。付与は同意と同じ保管先の別の行で、上の
	// クライアントの同意の操作には現れない。
	FindAgentDelegation(ctx context.Context, tenantID, sub, agentID string) (*consentdomain.Consent, error)
	// ListAgentDelegations は sub のすべての状態の付与を agent_id の昇順に返す。
	ListAgentDelegations(ctx context.Context, tenantID, sub string) ([]*consentdomain.Consent, error)
	// ListAgentDelegationPage は tenantID の付与を (user_id, agent_id) の昇順に、キーセットの
	// 後から limit 件まで返す。最初のページには "", "" を渡す。
	ListAgentDelegationPage(ctx context.Context, tenantID, afterUserID, afterAgentID string, limit int) ([]*consentdomain.Consent, error)
	ListAgentDelegationPageBefore(ctx context.Context, tenantID, beforeUserID, beforeAgentID string, limit int) ([]*consentdomain.Consent, error)
	// SaveAgentDelegation は付与を Granted で置き換える。last_used_at は保つ。
	SaveAgentDelegation(ctx context.Context, tenantID string, c *consentdomain.Consent) error
	RevokeAgentDelegation(ctx context.Context, tenantID, sub, agentID string) error
	// TouchAgentDelegation は付与の last_used_at を at にする。
	TouchAgentDelegation(ctx context.Context, tenantID, sub, agentID string, at time.Time) error
}
