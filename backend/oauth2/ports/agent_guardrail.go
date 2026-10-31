package ports

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// ErrAgentGuardrailConflict は保存の expected_version が現在のバージョンと異なることを表す。
var ErrAgentGuardrailConflict = errors.New("agent guardrail version conflict")

// AgentGuardrailPolicyRepository はガードレールの永続境界である。agentID が空ならテナントのデフォルトを指す。
type AgentGuardrailPolicyRepository interface {
	// ListEffective はテナントのデフォルトと、指定した Agent のポリシーを返す。
	ListEffective(ctx context.Context, tenantID string, agentIDs []string) ([]*domain.AgentGuardrailPolicy, error)
	Find(ctx context.Context, tenantID, agentID string) (*domain.AgentGuardrailPolicy, error)
	// Save は現在のバージョンが expectedVersion と等しいときだけ保存し、異なれば ErrAgentGuardrailConflict を返す。
	// ポリシーのない対象の現在のバージョンは空文字列である。
	Save(ctx context.Context, policy *domain.AgentGuardrailPolicy, expectedVersion string) error
	Delete(ctx context.Context, tenantID, agentID string) error
}

// IssuanceGuard は署名より前に発行を評価する。確定の前に拒否しなければならない経路が使う
// (REQ-OAUTH2-075)。同じ要求の SignAccessToken は Precheck の結果を再利用し、二重に数えない。
type IssuanceGuard interface {
	Precheck(ctx context.Context, in AccessTokenInput) error
	// Confirm は SignAccessToken を通らない発行 (ID-JAG) の評価を記録する。
	Confirm(tokenID string)
}
